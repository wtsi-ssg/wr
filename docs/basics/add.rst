Add
===

.. tip::
    Full details of all add's options can be found with ``wr add -h``.

Once the manager has been started, you can add the commands you want to get
executed to the manager's queue by using the add sub-command.

The simplest usage would be to place all the commands you need to execute in to
a text file, 1 command per line, and then::

    wr add -f commands.txt

.. tip::
    ``-f`` defaults to '-', which means read from STDIN, so you can also pipe
    commands in, which can be a quick way of adding a single command, eg::

        echo "myexe --arg a" | wr add

However, it would be best if you also specified additional information, which
are discussed in the following sections.

``wr add`` returns as soon as the jobs get successfully added to the queue, and
it tells you how many jobs were added. They will get scheduled and executed at
some point in the future, when the manager is able to find space amongst your
computing resources.

.. note::
    For just adding a single command, you can use the ``--sync`` option to 
    instead wait until the manager is able to get the command to either succeed
    or fail, whereupon ``wr add`` will exit with your command's exit code. 

Working directory
-----------------

By default, wr will create a unique sub directory in the directory you run
``wr add`` in, and execute your command from inside there. Also by default,
when your command finishes running, the unique sub directory will be deleted.

This is to aid isolation of your commands, so they don't interfere with each
other if you have many running simultaneously, and to clean up any temp files
your command creates. $TMPDIR will be set to a sister directory of the created
sub directory and deleted regardless of your cleanup options. It also lets wr
track how much disk space your command uses. Finally, you can use the
``change_home`` option to set $HOME to the working directory.

It does mean, however, that you need to specify any paths in your command line
using absolute paths, and any final output files you want to keep need to be
created in or moved to an absolute location (and you'll need to be careful
that multiple of your commands don't write to the some absolute final output
location).

Alternatively, you can specify ``--cwd_matters``, and your command will run
directly from the directory you ``wr add`` from (or the one you specify with
``--cwd``). You can then use relative paths. But there won't be any automatic
cleanup or disk space tracking. $TMPDIR and $HOME will be unchanged.

Rep_grp
-------

The commands you add have an identifer called a 'rep_grp', short for report
group. This is an abitrary name you can give your commands so you can easily
manipulate them later (eg. get the status of particular commands, or modify
them, or kill them etc.).

If you don't specify a rep_grp with the ``-i`` or ``--rep_grp`` option, then it
defaults to 'manually_added'.

It will be useful to you if you carefully consider a unique rep_grp to apply to
sets of similar commands, that is made up of sub-strings shared with related
commands.

For example, imagine you had a 2 step workflow designed to do the job of
'xyzing' your data, that first ran an executable ``foo`` on a particular input
file, then ran another command ``bar`` using foo's output as its input. 

If you then had input files '1.txt', '2.txt' and '3.txt', you could add the
commands for this workflow using rep_grps like::

    echo "foo 1.txt > 1.out.foo" | wr add -i 'xyz.step1.input1' --cwd_matters
    echo "foo 2.txt > 2.out.foo" | wr add -i 'xyz.step1.input2' --cwd_matters
    echo "foo 3.txt > 3.out.foo" | wr add -i 'xyz.step1.input3' --cwd_matters
    echo "bar 1.out.foo > 1.out.bar" | wr add -i 'xyz.step2.input1' --cwd_matters
    echo "bar 2.out.foo > 2.out.bar" | wr add -i 'xyz.step2.input2' --cwd_matters
    echo "bar 3.out.foo > 3.out.bar" | wr add -i 'xyz.step2.input3' --cwd_matters

.. note::
    In reality, it would be more efficient to add all these commands in one go,
    and you'd also need to specify :doc:`/advanced/dependencies` for this to
    work properly.

With this arrangement, you now have the flexability to manipulate eg.:

* a single job: ``-i 'xyz.step2.2'``
* all the step 2 jobs: ``-i 'xyz.step2' -z``
* all the jobs that manipulated input 2: ``-i 'input2' -z``
* all the jobs in the xyz workflow: ``-i 'xyz' -z``

.. _resource-usage-learning:

Resource Usage
--------------

To make the most efficient use of your available hardware resources, you should
specify how much time, memory, disk and CPU your commands will use. With this
knowledge, wr will be able to schedule as many of your commands to be run at
once, without overloading any particular machine.

``--memory`` and ``--time`` let you provide hints to wr manager so that it can
do a better job of spawning runners to handle these commands. "memory" values
should specify a unit, eg "100M" for 100 megabytes, or "1G" for 1 gigabyte.
"time" values should do the same, eg. "30m" for 30 minutes, or "1h" for 1 hour.

``--cpus`` tells wr manager exactly how many CPU cores your command needs. CPU
usage is not learnt.

``--disk`` tells wr manager how much free disk space (in GB) your command needs.
Disk space reservation only applies to the OpenStack scheduler which will
create temporary volumes of the specified size if necessary. Note that disk
space usage checking and learning only occurs for jobs where cwd doesn't matter
(is a unique directory), and ignores the contents of mounted directories.

.. note::
    By default, wr will assume 1GB memory, 1hr, 0GB disk and 1CPU per command.

The manager learns how much memory, disk and time commands in the same
``--req_grp`` actually used in the past, and will use its own learnt values
unless you set an override. For this learning to work well, you should have
reason to believe that all the commands you add with the same req_grp will have
similar memory and time requirements, and you should pick the name in a
consistent way such that you'll use it again in the future.

For example, if you want to run an executable called "exop", and you know that
the memory and time requirements of exop vary with the size of its input file,
you might batch your commands so that all the input files in one batch have
sizes in a certain range, and then provide a req_grp that describes this, eg.
"exop.1-2Ginputs" for inputs in the 1 to 2 GB range.

.. note::
    Don't name your req_grp after the expected requirements themselves, such as
    "5GBram.1hr", because then the manager can't learn about your commands - it
    is only learning about how good your estimates are! The name of your
    executable should almost always be part of the req_grp name.

    req_grp defaults to the first word in your cmd, which will typically be the
    name of your executable.

``--override`` defines if your memory, disk or time should be used instead of
the manager's estimate. Possible values are:

* 0 = do not override wr's learned values for memory, disk and time (if any)
* 1 = override if yours are higher
* 2 = always override specified resource(s)
  
.. note::
    If you choose to override eg. only disk, then the learned value for memory
    and time will be used. If you want to override all 3 resources to disable
    learning completly, you must explicitly supply non-zero values for memory
    and time and 0 or more for disk.)

.. _job-priority:

Retries
-------

If your command exits non-0, ``--retries`` defines how many times it will be
retried automatically until it succeeds.

Automatic retries are helpful in the case of transient errors, or errors due to
running out of memory or time (when retried, they will be retried with more
memory/time reserved).

Once this number of retries is reached, the command will be 'buried' until you
take manual action to fix the problem and press the retry button in the web
interface or use :doc:`wr retry <retry>`.

.. note::
    By default, there will be 3 retries.

When a command fails, if there are retries remaining, before the command is run
again there will be a delay, and the length of the delay depends on the number
of attempts so far, increasing from 30s by a factor of 2 each attempt, up to a
maximuim of 1hr. The delay time is also jittered by up to 30s, to avoid the
thundering herd problem.

``--no_retry_over_walltime`` defines a time which if a command runs longer than
and fails, it will be immediately buried, regardless of the "retries" value.
This is useful for commands that might fail quickly due to some transient
initialization issue, and would likely succeed if retried, but are always
expected to fail if they get past initialization and then fail. The default
value of 0 time disables this feature and jobs will always retry according to
``--retries``.

Priority
--------

You can influence the order that the commands you add to the queue get executed
using the ``--priority`` option.

This defines how urgent a particular command is; those with higher priorities
will start running before those with lower priorities. The range of possible
values is 0 (default, for lowest priority) to 255 (highest priority).

Commands with the same priority will be started in the order they were added.

.. note::
    However, that order of starting is only guaranteed to hold true amongst jobs
    with similar resource requirements, since your chosen job scheduler may, for
    example, run your highest priority job on a machine where it takes up 90% of
    memory, and then find another job to run on that machine that needs 10% or
    less memory - and that job might be one of your low priority ones.

Dependencies
------------

By default, the manager will try to get all the commands you add to the queue
running at once, assuming there is enough capacity in your compute environment.
That means if have a command Y that should only run after a command X has
succeeded, and you add both X and Y to the queue, the manager could end up
running them at the same time, and Y would presumably fail.

To specify that certain commands should only be executed after certain other
ones have completed, you can place commands in ``--dep_grps`` and then have
other commands be dependant upon those ``--deps``.

For an in-depth look at dependencies, see :doc:`/advanced/dependencies`.

Limiting
--------

By default, the manager will try to get all the commands you add to the queue
running at once, assuming there is enough capacity in your compute environment,
and dependencies have been met.

If you have a command that interacts with some limited resource (eg. a database
with a maximum number of client connections allowed), you can tell the manager
to limit how many of those commands to run simultaneously by placing them in the
same limit group.

``--limit_grps`` is a comma separated list of arbitrary names you can associate
with a command, that can be used to limit the number of jobs that run at once in
the same group. You can optionally suffix a group name with :n where n is a
integer new limit for that group. 0 prevents jobs in that group running at all.
-1 makes jobs in that group unlimited. If no limit number is suffixed, groups
will be unlimited until a limit is set with the :doc:`wr limit <limit>` command.

.. tip::
    Use :doc:`wr limit <limit>` to change your limits after adding jobs.

For example, if you had a database that only allowed 100 connections, but you
had 1000 different commands that needed to access the database, you could put
all 1000 commands in a text file and then::

    wr add -f db.cmds --limit_grps 'mydb:100'

The manager would only schedule up to 100 of these commands to run at once. If
you had commands that accessed both your database and a very slow archival disk
that could only handle 5 writes at once, you could::

    wr add -f archive.cmds --limit_grps 'mydb,archive:5'

The manager would schedule none of these jobs until the first 905 database-only
jobs in this example had completed, then would only run 5 of these archive jobs
at once. If you then added more database-only jobs before these archive jobs
completed, 95 of them would run at once, alongside the 5 archival jobs.

Behaviours
----------

You can associate certain behaviours with the commands you add. Behaviours are
triggered when your command exits, and run from the same working directory.

There are 3 variations on the trigger:

``--on_failure``
    Behaviours trigger if your command exits non-0.

``--on_success``
    Behaviours trigger if your command exits 0.

``--on_exit``
    Behaviours trigger when your command exits, regardless of exit code. These
    behaviours trigger in addition to and after any on_failure or on_success
    behaviours.

Behaviours are described using an array of objects, where each object has a key
corresponding to the name of the desired behaviour, and the relevant value. The
currently available behaviours are:

"cleanup"
    Takes a boolean value which if true will completely delete the actual
    working directory created when cwd_matters is false (no effect when
    cwd_matters is true). This behaviour is by default turned on for the
    on_exit trigger.

    .. tip::
        You can disable the default cleanup behaviour by saying
        ``--on_exit '[]'``

"run"
    Takes a string command to run after the main cmd runs.

"remove"
    Takes a boolean value which if true means that if the cmd gets buried, it
    will then immediately be removed from the queue (useful for Cromwell
    compatibility).

For example
``--on_exit [{"run":"cp warn.log /shared/logs/this.log"},{"cleanup":true}]``
would copy a log file that your cmd generated to describe its problems to some
shared location and then delete all files created by your cmd.

Mounts
------

If your command needs input or output files in an S3-like object store, it will
be convienent and probably faster and more efficient to use wr's built-in
high-performance S3 fuse mounting capability. (As opposed to manually
downloading or uploading files with another tool.)

For details on how to use S3 with wr, read :doc:`this guide </advanced/s3>`.

Your mounts will be unmounted after the triggering of any behaviours, so your
"run" behaviours will be able to read from or write to anything in your mount
point(s). The "cleanup" behaviour, however, will ignore your mounted directories
and any mount cache directories, so that nothing on your remote file systems
gets deleted. Unmounting will get rid of them though, so you would still end up
with a "cleaned" workspace.
