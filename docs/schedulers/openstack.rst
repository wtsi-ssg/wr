OpenStack
=========

The OpenStack scheduler lets you add jobs to wr's queue and have wr execute them
in an OpenStack envionment. It will create OpenStack instances of appropriate
flavors to cope with the amount of work you have to do (auto scale-up), and
destroy them when your work completes (auto scale-down).

It uses bin-packing to potentially fit multiple jobs on an instance at once, and
re-uses instances if there's more work to do.

Usage
-----

Using wr with OpenStack requires that you source your openstack RC file. Log in
to your OpenStack dashboard web interface and look for the 'Download Openstack
RC File' button. For older installs this is in the Compute -> Access & Security,
'API Access' tab. For newer installs it is under Project -> API Access. You may
need to add to that RC file OS_POOL_NAME to define the name of the network to
get floating IPs from (for older installs this defaults to "nova", for newer
ones it defaults to "public").

If you're able to create your own OpenStack network, keys, security profiles and
instance, ssh to your OpenStack instance, install wr there, copy your RC file
there, source it and then::

   wr manager start -s openstack --local_username yourname

.. note::
   See ``wr manager start -h`` for various other options you may need to set,
   in particular all the '--cloud*' options.

When this manager needs to create new OpenStack instances to run your commands
on, it will create them in the same network and with similar security settings
as the instance you start the manager on.

An easier alternative to this, especially if you don't know how to use
OpenStack, is to have wr deploy to OpenStack from a machine that has API access
to your OpenStack environment (somewhere that the ``openstack`` commands work).

First source your RC file, then::

   wr cloud deploy

.. note::
   Again, see ``wr cloud deploy -h`` for various other options you may need to
   set, which are like the manager's '--cloud*' options, but without the 'cloud'
   prefix.

   ``wr cloud deploy`` has a default ``--os``, but it may not be suitable for
   your particular installation of OpenStack. Don't forget that you can change
   the default by setting 'cloudos' in your :doc:`config </basics/config>`.

The cloud deploy will create the various required OpenStack resources such as
network, key and instance, copy your wr executable to the instance, and start
the manager there. It will also set up some ssh port forwarding, so that you can
use the other wr commands on the local machine you deploy from, and they will
communicate with the manager on the OpenStack instance.

When you've finished your work, you should delete the resources wr created by
doing::

   wr cloud teardown

After you teardown, the manager's log file can be found on the local machine you
deployed from in (by default) `~/.wr_production/log.openstack`.

Using cloud deploy is also the best way to ensure complete recovery of state in
the event of a disaster. See :doc:`/advanced/recovery>` for more details. With
the 'managerdbbkfile' config option pointing to a location in S3, if for some
reason the deployed instance where wr is running goes bad and gets destroyed,
you can just deploy again and recover state.

Follow the :doc:`OpenStack tutorial </guides/openstack>` for an example of doing
some real work in OpenStack.

The private key
---------------

wr will create a key in OpenStack that is used to ssh to the instances it
creates. The key is named after the '--local_username' option you supply to
``wr manager start -s openstack``, which is set to your actual username when
using ``wr cloud deploy``.

It's important that this name is globally unique, because if a key with that
name already exists, wr will not create a new one and instead try to re-use it,
but won't have the actual private key on disk, so won't be able to ssh to any
instances it creates.

.. tip::
   The most common cause of failure to cloud deploy or start the manager on an
   OpenStack instance is a problem with the private key already existing. A ``wr
   cloud teardown`` or ``wr manager stop`` should normally delete the key, but
   something might go wrong and a key gets left behind. Use ``openstack keypair
   list`` and ``openstack keypair delete`` to delete your wr keypair before
   trying again.

The private key is by default stored in
``~/.wr_production/cloud_resources.openstack.key`` on both the machine you
deploy from and the OpenStack instance the manager runs on, so you can supply it
to ``ssh -i`` to manually ssh to the instances wr creates should you need to
investigate something.

Docker users
------------

If you use docker, you will have to configure it to not conflict with your local
system's network or the network that wr will create for you. For example, the
script you supply to ``wr cloud deploy --script`` might start::

   sudo mkdir -p /etc/docker/
   sudo bash -c "echo '{ \"bip\": \"192.168.3.3/24\", \"dns\": [\"8.8.8.8\",\"8.8.4.4\"], \"mtu\": 1380 }' > /etc/docker/daemon.json"
   [further commands for installing docker]

Sanger Institute users
----------------------

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