Schedulers
==========

When you add jobs to wr, the manager schedules those jobs to be run in your
compute environment using the requested job scheduler.

There are currently 3 different schedulers to pick from:

* :doc:`Local </schedulers/local>` for executing on just your local machine.
* :doc:`LSF </schedulers/lsf>` for executing on an LSF cluster.
* :doc:`OpenStack </schedulers/openstack>` for executing on an OpenStack cluser.

Implementation details
----------------------

What actually happens under the hood after you add jobs?

1. When you run ``wr add``, your commands become "jobs" that are added to a job
   queue, with a reservation group based on your command's resource
   requirements (the "size" of the job: how much memory, cpu and time it is
   expected to consume). This job queue is held in memory by the manager.

   .. note::
      As a failsafe, the state of jobs is also recorded on disk in an ACID
      transactional embedded database, which is also where completed jobs are
      stored permanently.

2. The manager constantly takes these actions in the background:

    1. When jobs become "ready" in the queue (eg. as soon as they're first
       added), count how many are in each reservation group (possibly altering
       that first if :ref:`resource-usage-learning` is in place), and ask the
       scheduler to schedule that many ``wr runner --server <ip of manager's
       host:manager's port> --scheduler_group <reservation group>`` commands.
    2. In case something silently goes wrong with the scheduler or runners,
       arrange to force a repeat of this after 1min even if no new jobs get
       added (working on jobs still "ready" in the queue at that time).

3. It is the scheduler's job to execute the ``wr runner`` process the given
   number of times, in the most appropriate way.
   
   .. note::  
      A scheduler doesn't guarantee that the ``wr runner`` processes *succeed*
      the given number of times, only that they will be *started* that many
      times.
      
      A scheduler also handles the number increasing (eg. if more of a certain
      sized job are added to the queue) or decreasing (eg. if the scheduler had
      queued up jobs because they couldn't all run at once, but now you removed
      jobs or some completed).

      The benefit of this system is that there may be scheduling advantages to
      grouping same-sized jobs (even if they are for completely different kinds
      of commands), for example with the LSF scheduler wr can create large
      efficient job arrays.

4. When a ``wr runner`` process starts running somewhere, it takes these
   actions:

   1. Connect to the manager, using the knowledge of its IP:port.
   2. Reserve the next highest priority job in the manager's job queue that has
      this runner's ``--scheduler_group`` as its reservation group.
   3. Run the command line, send a message to the manager to note that the job
      has started to run, keep track of memory and disk usage, and regularly
      contact the manager to "touch" the job being executed.
   4. When the command exits, tell the manager about its final state.
   5. Reserve another job and loop back to 3. Exit if there are no more jobs in
      the group after waiting for 2 seconds.

5. Back on the manager, when a runner reserves a job, it becomes "reserved" in
   the manager's queue, which starts a countdown. When the runner sends the
   message that it started to run the command, it becomes "running" in the
   manager's queue. When the countdown finishes, the job is considered "lost".
   The countdown is restarted every time the runner "touches" the job.

   .. note:: 
      When a job is lost, if the runner manages to eventually touch it again
      (eg. it couldn't during the countdown due to a temporary networking
      issue), it will go back to "running" state automatically.

      When lost, you will see it lost in status and can investigate. If it is
      permanently lost (eg. the OpenStack server it was running on simply
      doesn't exist anymore, or has suffered a kernel panic or similar), you can
      confirm the job is dead, which buries it. Then you can retry when desired.

These 5 steps ensure:

* Jobs get scheduled the moment you add them.
* They are efficiently grouped together by size for scheduling.
* The manager, scheduler and the runners can all fail, but:
  
  * Jobs are never lost from the job queue.
  * Jobs that the scheduler doesn't manage to schedule correctly will be retried
    until they do get scheduled.
  * You'll know about jobs that started to run but aren't actually running
    anymore, and can do something about them (with automatic retries possible).

* The manager knows the moment jobs complete (even before the scheduler knows),
  how long they took to run, and their cpu and memory usage (even on schedulers
  that don't track these things). This allows dependent jobs to be immediately
  scheduled, or the system to be scaled down.