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
your command creates. It also lets wr track how much disk space your command
uses.

It does mean, however, that you need to specify any paths in your command line
using absolute paths, and any final output files you want to keep need to be
created in or moved to an absolute location (and you'll need to be careful
that multiple of your commands don't write to the some absolute final output
location).

Alternatively, you can specify ``--cwd_matters``, and your command will run
directly from the directory you ``wr add`` from (or the one you specify with
``--cwd``). You can then use relative paths. But there won't be any automatic
cleanup or disk space tracking.

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
    and you'd also need to specify :ref:`job-dependencies` for this to work
    properly.

With this arrangement, you now have the flexability to manipulate eg.:

* a single job: ``-i 'xyz.step2.2'``
* all the step 2 jobs: ``-i 'xyz.step2' -z``
* all the jobs that manipulated input 2: ``-i 'input2' -z``
* all the jobs in the xyz workflow: ``-i 'xyz' -z``

.. _resource-usage-learning:

Resource Usage Learning
-----------------------

docs coming soon...

.. _job-priority:

Priority
--------

docs coming soon...

.. _job-dependencies:

Dependencies
------------

docs coming soon...

For an in-depth look at dependencies, see :doc:`/advanced/dependencies`.