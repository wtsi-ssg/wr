OpenStack
=========

coming soon...

Using wr with OpenStack requires that you source your openstack rc file; see wr
cloud deploy -h.

.. note::
   ``wr cloud deploy`` has a default ``--os``, but it may not be suitable for
   your particular installation of OpenStack. Don't forget that you can change
   the default by setting 'cloudos' in your :doc:`config </basics/config>`.



   managerdbbkfile: "s3://bucket/wr_backups/db_bk"

The managerdbbkfile pointing to a location in S3 means that if for some reason
the deployed instance where wr is running goes bad and gets destroyed, you can
just deploy again and recover state)


Docker users
^^^^^^^^^^^^

If you use docker, you will have to configure it to not conflict with your local
system's network or the network that wr will create for you. For example, the
script you supply to ``wr cloud deploy --script`` might start::

   sudo mkdir -p /etc/docker/
   sudo bash -c "echo '{ \"bip\": \"192.168.3.3/24\", \"dns\": [\"8.8.8.8\",\"8.8.4.4\"], \"mtu\": 1380 }' > /etc/docker/daemon.json"
   [further commands for installing docker]

Sanger Institute users
^^^^^^^^^^^^^^^^^^^^^^

If you're at the Sanger Institute and want to use wr with OpenStack, you'll need
to use a flavor regex of something like::

   ^[mso].*$

You'll probably also want to use Sanger's DNS IPs, to resolve local domains.

It'll be easiest if you set these and other cloud options in your config file
(``~/.wr_config.yml``)::

   cloudflavor: "^[mso].*$"
   cloudflavorsets: "s2;m2;m1;o2"
   clouddns: "172.18.255.1,172.18.255.2,172.18.255.3"
   # (the following are the defaults and don't need to be set)
   cloudcidr: "192.168.0.0/18"
   cloudgateway: "192.168.0.1"
   cloudos: "bionic-server"
   clouduser: "ubuntu"
   cloudram: 2048

   And your `~/.wr_config.yml` might look like:

Multiple deployments
--------------------

Normally you can do a single ``wr cloud deploy --deployment production``, and a
single ``wr cloud deploy --deployment development``. This should be fine if
you're a normal single user running workflows for yourself. Other people can do
their own deployments from the same machine and you won't have any conflicts.

If, however, you have full control of a machine and want to run multiple
deployments yourself (eg. you're running wr for other people, but want to keep
each of their workflows in separate cloud networks), you can do something like::

   export MY_UNIQUE_DEPLOYMENT_NAME="one"
   export WR_MANAGERPORT=11320
   export WR_MANAGERWEB=11321
   export WR_MANAGERDIR="~/.wr_$MY_UNIQUE_DEPLOYMENT_NAME"
   wr cloud deploy --resource_name $MY_UNIQUE_DEPLOYMENT_NAME
   [use wr commands as normal]
   wr cloud teardown --resource_name $MY_UNIQUE_DEPLOYMENT_NAME

You will have to arrange that the value of ``--resource_name`` is unique within
your cloud, and that WR_MANAGERPORT and WR_MANAGERWEB are unique (and not used
by anyone else) on your machine, for each deployment you want to do.

Implementation details
----------------------

The OpenStack scheduler is based on the :doc:`local scheduler <local>`, using
its special queue processing.

Processing the queue is modified to first check, at most every 1 min, that all
OpenStack servers we have spawned and expect to be up, can still be ssh'd to,
trigger user warnings if not (and remove user warnings if things get better).

The standard queue processing loop is also altered to take in to account known
quota limits when deciding what will fit into remaining resources. 

To run a command with Y resource requirements: *the following is out-dated*

1. Check existing ssh-able servers to see if they have space for Y.
2. If 1 has, allocate that Y resources are now used on that server, and run the
   command on that server in an ssh session.
3. Otherwise, see if there's space on a sever we have queued to spawn, and if so
   assign it to that "standin" server.
4. Otherwise, figure out the cheapest flavor that supports Y, and spawn a new
   server, except that we wait for prior spawn requests to complete first,
   forming a spawn queue.
5. When a server has spawned (but not yet finished powering up), trigger the
   spawning of the next server in the queue.
6. Wait for boot up, run user boot script, copy over wr executable.
7. Allocate this job and any standin jobs to this server, then run the command
   via an ssh session.

When all the commands it ran via ssh sessions on a spawned server exit (ie. when
all those runners can no longer find matching jobs in the manager's queue), a
countdown is started (of length equal to the --cloud_keepalive argument to ``wr
manager start``), and on completion the server is destroyed, which is how scale
down occurs.

When a single command exits, the scheduler deallocates Y resources from it and
triggers a processing of the queue, which will now see the free space on the
server and have the potential to run a new command there.