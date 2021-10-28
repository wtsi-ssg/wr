Manager
=======

The wr manager works in the background, doing all the work of ensuring your
commands get executed.

It maintains both a temporary queue of the commands you want to run, and a
permanent history of commands you've run in the past. As commands are added to
the queue, it makes sure to spawn sufficient 'wr runner' agents to get them all
executed.

It won't run the exact same command more than once at a time, helping avoid
corrupted command output if the same command was to write to the same output
file multiple times at once.

It guarantees that your commands either complete successfully, or that you know
about and can then do something about commands that fail.

Starting
--------
::

    wr manager start

.. note::
    With no other options or configuration, this will use sensible defaults and
    execute your commands on the local machine. You may need to first alter your
    :doc:`configuration </basics/config>`, or use a different
    :doc:`scheduler </schedulers/schedulers>`.

When you start the manager, it will output information like::

    INFO[10-28|08:45:04] wr manager v0.28.0-0-ga48ed0d started on 172.27.71.219:46407, pid 5425
    INFO[10-28|08:45:04] wr's web interface can be reached at https://localhost:46408/?token=DPPOGNjt5KJJ7sAJzJNrv2YGUSQV82Ad5Us7St50FmE

The first line there tells you the version of wr you're using, and the IP
address and port that the other wr sub-commands will connect to in order to add
or manipulate jobs in the manager's queue. It also shows you the pid of the
manager process.

The second line gives you the URL for the status web interface, where you can
see real-time updates on the status of the jobs you add to the queue.
The port number and token in the URL are also those that are needed to access
the REST API.

.. note::
    When viewing the web interface, your browser will raise a security warning,
    since by default wr will generate and use its own self-signed certificate.
    So the first time you view it you will need to allow an exception. See the
    section on :doc:`security </advanced/security>` to learn how to use your own
    certificates instead.

.. note::
    If you are ssh tunnelling to the node where you start the manager and wish
    to use the web interface, you will have to forward the host and port that it
    tells you the web interface can be reached on. The following example
    ``~/.ssh/config`` file may help. (In this example, 46408 is the web
    interface port that wr tells you about, domain is your domain name,
    ss.domain is the host you ssh to, and manager_node.internal.domain is the
    host you start the manager on.)

    .. code-block:: console
        :name: ssh-forwarding-example

        ControlMaster auto
        ControlPath ~/.ssh/controlpaths/s_%C

        Host ssh.domain
            User username
            DynamicForward 1080
            ProxyCommand none

        Host *.internal.domain
            User username
            ForwardAgent yes
            ProxyJump ssh.domain

        Host manager_node.internal.domain
            HostName manager_node.internal.domain

    You'll then be able to access the website at
    https://manager_node.internal.domain:46408 after doing ``ssh -Nf
    ssh.domain``.

.. _manager-deployments:

Deployments
-----------
Without having to change any configuration on-the-fly, you can run 2 instances
of the manager at once: a production deployment, and a development deployment.

This is possible because the different deployments can be configued to use
different ports and manager directories (and thus different database and log
file locations etc.). This is handled for you by the default configuration
calculating different ports depending on deployment. The deployment is also
appended to the directory name you specify for the 'managerdir' option.

Other than using different ports and directories (and thus different databases
etc.), the key difference between the deployments is that the production
deployment maintains a permanent history of all the commands you ever add in its
database, while the development deployment deletes its database and creates a
new one every time you start the manager.

If you need more than 1 production deployment running at once, eg. you want a
single "service" user to run jobs for multiple other "real" users, then you
should change these environment variables to switch between the deployments,
being sure to change all them before using any wr command to interact with that
deployment::

    export MY_UNIQUE_DEPLOYMENT_NAME="one"
    export WR_MANAGERPORT=11320
    export WR_MANAGERWEB=11321
    export WR_MANAGERDIR="~/.wr_$MY_UNIQUE_DEPLOYMENT_NAME"

You will have to arrange that the value of MY_UNIQUE_DEPLOYMENT_NAME is unique
within your compute environment, and that WR_MANAGERPORT and WR_MANAGERWEB are
unique (and not used by anyone else) on your machine, for each deployment you
want to do.

.. note::
    If you're doing cloud deployments, after setting the environment variables
    be sure to also specify '--resource_name', eg: ``wr cloud deploy
    --resource_name $MY_UNIQUE_DEPLOYMENT_NAME`` and ``wr cloud teardown
    --resource_name $MY_UNIQUE_DEPLOYMENT_NAME``.

Stopping
--------
You can leave the manager daemon running indefinitely, so it's ready to accept
new jobs whenever you need to add them. 

But if you no longer need the manager to be running, you can cleanly shut it
down by doing::
    
    wr manager stop
    
You can also ``kill <pid>``, where '<pid>' is the pid of the manager process,
which you were told about when the manager started.

A clean shutdown will kill any currently running jobs, wait for them to exit,
wait for the scheduler to clean up any used resources, and then the manager
daemon will exit.

The next time you start the manager, with a development deployment you will be
in a brand-new state, having lost any record of the jobs that may have been
previously running or completed.

With a production deployment, your prior state is recovered, with any jobs that
were killed now in 'buried' state, any jobs that were pending now eligble to
start running, and completed jobs searchable in the history and available as
dependencies.

To force the manager to exit immediately without cleanly shutting down, do
``kill -9 <pid>``. This will leave running jobs to continue running, and
scheduler resources will continue to be used. (The same applies if the manager
process were to crash and exit by itself.)

With a production deployment, if you then start the manager again within 24hrs,
the jobs that were running will reconnect to the new manager and everything will
continue as if you hadn't killed the manager. See :doc:`/advanced/recovery` for
more details.

.. tip::
    Force killing the manager can thus be a useful way of updating to a new
    version of wr without having to interrupt your work.

.. note::
    If jobs were running when the manager process was force killed or crashed,
    and they finish running while the manager is offline, you have 24hrs to
    start the manager again; if you do so then their completed state will be
    recorded and things will continue normally. If more than 24hrs pass,
    however, the fact that the commands completed will not be known by the new
    manager, and they will eventually appear in "lost contact" state. You will
    have to then confirm them as dead and retry them from the start (even though
    they had actually completed).

In-between the clean shutdown and force killing, there is draining::

    wr manager drain

This will keep the manager running for as long as there are running jobs,
without starting to run any new jobs. New jobs can still be added to the queue;
they just won't be scheduled for execution.

Repeatedly running the drain command will give you an estimate on how long it
will be before the currently running jobs will complete, and the manager will
stop itself.
