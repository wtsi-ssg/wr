Config
======

.. tip::
    ``wr conf -h`` explains configuration as well.

Running ``wr conf --default`` will print an example config file which details
all the config options available.

The default config should be fine for most people, but if you want to change
something, run ``wr conf --default > ~/.wr_config.yml`` and make changes to
that (or save it as one of the other possible
:ref:`config files <config-files>`). Alternatively, export
:ref:`environment variables <env-vars>`.

.. tip::
    If you'll be using OpenStack, it is strongly recommended to configure that
    database backups go to S3 by setting
    ``managerdbbkfile: s3://yourbucket/yourpath``.

Running just ``wr conf`` will show you your currently configured values, and
remind you where that configurtion is set (which files, or if it is set in an
environment variable, or if it's a default).

.. _config-files:

Config files
------------
wr will load its configuration settings from one or more files named
.wr_config[.production|.development].yml found in these directories, in order of
precedence:

1. The current directory
2. Your home directory
3. The directory pointed to by the environment variable $WR_CONFIG_DIR

``.wr_config.yml`` files are always read, and can be used to define settings
common to both production and development :ref:`deployments
<manager-deployments>`.

``.wr_config.development.yml`` files are only read in a development context:
    Either a ``--deployment development`` option has been passed to the wr
    executable, or the environment variable $WR_DEPLOYMENT has been set to
    'development'. Or the current working directory contains a checkout of the
    wr git repository.

``.wr_config.production.yml`` files are only read in a production context:
    Either a ``--deployment production`` option has been passed to the wr
    executable, or the environment variable $WR_DEPLOYMENT has been set to
    'production'. Or we are not in a development context, where we default to
    production.

.. note::
    If you use config files, these must be readable by all nodes; when you don't
    have a shared disk, it's best to configure using environment variables.
    
    In cloud deployments where wr itself creates compute nodes, a config file
    will be created on new nodes automatically.

.. _env-vars:

Environment variables
---------------------

You can set and override config settings by defining environment variables named
like: ``WR_<setting name in caps>``. Eg. to define the ``managerscheduler``
option you might do::

    export WR_MANAGERSCHEDULER="lsf"

.. note::
    Environment variables will apply to both deployments, so you shouldn't set
    options that must be different between them, such as :ref:`ports
    <manager-ports>`, unless you're careful to also change WR_DEPLOYMENT at the
    same time, or you will only ever be running one of the deployments.

.. _manager-ports:

Ports
-----
The wr manager needs 2 ports to operate, one for the wr executable, one for the
web interface and REST API. By default, it will calculate 4 ports based on your
uid (2 for each deployment), so that different people starting the manager on
the same machine will get different ports.

This calculation will fail if you have a very high uid. In this case, when you
start the manager it will prompt you to accept random ports that are currently
available, and write these in to your config file.

Alternatively you could manually set ``managerport`` and ``managerweb`` in your
config files (you must use different ones for your development and production
deployments) to desired port numbers yourself.

.. note::
    If you use an integration like Nextflow where it by default guesses your
    ports, you'll need to instead configure Nexflow to use the same ports that
    are now specified in your wr config file.

Performance considerations
--------------------------
For the most part, you should be able to throw as many jobs at wr as you like,
running on as many compute nodes as you have available, and trust that wr will
cope. There are no performance-related parameters to fiddle with: fast mode is
always on!

However you should be aware that wr's performance will typically be limited by
that of the disk you configure wr's database to be stored on (by default it is
stored in your home directory), since to ensure that workflows don't break and
recovery is possible after crashes or power outages, every time you add jobs to
wr, and every time you finish running a job, before the operation completes it
must wait for the job state to be persisted to disk in the database.

This means that in extreme edge cases, eg. you're trying to run thousands of
jobs in parallel, each of which completes in milliseconds, each of which want to
add new jobs to the system, you could become limited by disk performance if
you're using old or slow hardware.

You're unlikely to see any performance degradation even in extreme edge cases
if using an SSD and a modern disk controller. Even an NFS mount could give more
than acceptable performance.

But an old spinning disk or an old disk controller (eg. limited to 100MB/s)
could cause things to slow to a crawl in this edge case. "High performance" disk
systems like Lustre should also be avoided, since these tend to have incredibly
bad performance when dealing with many tiny writes to small files.

.. tip::
    Set the ``managerdbfile`` option in your config file to a path on a fast
    disk.

If this is the only hardware you have available to you, you can half the impact
of disk performance by reorganising your workflow such that you add all your
jobs in a single `wr add` call, instead of calling `wr add` many times with
subsets of those jobs.