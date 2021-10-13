Basic Tutorial
==============

Why use wr?
-----------
wr is a job queue, where the "jobs" are command lines you wish to run.
You add jobs to its queue, and then wr will take care of executing your commands
on your available computing resources in the most efficient way possible.

If you only had a few command lines to run, you might just run them directly,
without the need for wr. wr becomes useful when you need to:

* Run hundreds or thousands of commands.
* Have some commands run only after others have completed successfully (you have
  dependencies).
* Prioritise certain commands.
* Only run a particular command line once at any given moment in time, so that
  2 or more attempts to run the same command don't overwrite and corrupt each
  others outputs.
* Run more commands than will fit on available hardware all at once.
* Run commands efficiently and easily over a cluster of compute nodes.
* Limit how many of a certain kind of command can run simultaneously (eg. if
  you have a command that accesses a limited external resource).
* Easily handle some commands failing: quickly identify the failures, look at
  the errors, (automatically) re-run them to get them to succeed (eg. after
  fixing some external issue, or altering the command).
* Monitor the progress of executing all your commands.
* Know when all your commands have completed sucessfully, and be confident that
  nothing slipped through the cracks and didn't get run at all, or failed
  without the failure being noticed.
* Summarise how long various different commands take to run and how much memory
  they use.

Outline usage
-------------
After :doc:`installation </basics/install>`, you'll have a ``wr`` command which
has numerous sub-commands.

.. tip:: Use the ``-h`` option on any wr sub-command to get detailed help text.

The first thing you need to do is start the wr "manager". This is a server
daemon that runs in the background, providing the job queue, and triggering the
execution of jobs. ::

    wr manager start

.. note::
    With no other options or configuration, this will use sensible defaults and
    execute your commands on the local machine. You may need to alter your
    :doc:`configuration </basics/config>`, or use a different
    :doc:`scheduler </schedulers/schedulers>`.

Once the manager is up and running, the other wr sub-commands act as clients
that communicate with the manager to do various things.

The next thing you'll need to do is add jobs to the queue. The simplest way of
doing this is to put all the command lines you want to run in a text file (1
command per line), and then::

    wr add -f cmds_in_a_file.txt

.. note::
    ``wr add`` had lots of options and there are better ways of using it than
    simply adding all your commands like this without specifying any other
    information about them; read the help text or the docs
    :doc:`here </basics/add>`.

Once you've done that you can:

* Monitor the :doc:`status </basics/status>` of your commands.
* :doc:`Retry </basics/retry>` failed commands.
* :doc:`Kill </basics/kill>` running commands.
* :doc:`Remove </basics/remove>` queued commands.
* :doc:`Modify </basics/mod>` queued commands.
* Change the :doc:`limits </basics/limit>` applied to certain commands.

.. note::
    There is a ``wr runner`` sub-command that you shouldn't run yourself. The
    manager executes runners on your compute resources, and the runners pick up
    jobs from the queue and then in turn execute your commands. See
    :doc:`schedulers </schedulers/schedulers>` for more info.

Finally, when your work is complete, you might::
    
    wr manager stop

(Though it isn't necessary to stop the manager; you can just leave it running
forever, so it's ready the next time you want to add jobs.)
