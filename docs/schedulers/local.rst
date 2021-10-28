Local
=====

The local scheduler is the default scheduler, and will execute your commands on
the local host directly. ::

    wr manager start -s local

It uses bin-packing to try and fit as many of your commands as possible on to
your machine at once.

This does mean that "larger" jobs (those that use more cpu and memory) will be
scheduled first, and potentially start running before smaller jobs that had
higher user-supplied :ref:`job-priority`. However, for jobs of the same size,
your own assigned priority will take effect.

To limit the amount of local resources used to execute your commands, you can
supply the max_cores and max_ram arguements, where -1 (the default) means
unlimited. For example, to limit to 2 cpus and 1GB RAM::

    wr manager start -s local --max_cores 2 --max_ram 1024

.. note::
    Specfying ``--max_cores 0`` will still allow 0-core jobs to be run. If you
    set ``--max_ram 0``, then nothing would run because you can't add 0MB jobs,
    which wouldn't be very useful!

Implementation details
----------------------

The local scheduler has its own independent queue. When a schedule request comes
in from the manager, it queues that request in the form "this command needs to
be run X times, and needs Y resources", updating X if a new schedule request
comes in for a command already in the queue (or if fewer are now needed because
some commands completed). Recall that "this command" is a ``wr runner`` command,
not one of the commands you added with ``wr add``.

The priority of items in the queue is based on how "large" Y is: the max of the
percentage of available memory it needs and percentage of cpus it needs.

It takes these actions:

1. Fail the schedule request if Y resources do not exist in the whole system.

2. Start an automated process that regularly (every 1min) processes the queue.

3. Process the queue in response to 2), or a new schedule request (if not
   currently processing; if we are, arrange that the processing will be
   immediately triggered once after current processing ends).

Processing the queue involves looping through commands in the queue in priority
order:

1. End loop if we've considered every command in the queue.
2. If we are already running >=X of this command, loop again.
3. Figure out how many Y sized things can fit into remaining resources, based on
   how much we're currently using or have allocated that we will use.
4. If there's no more space for this Y, loop again.
5. Initiate the running of this command X times.
6. Wait for each run initiation to allocate or actually use Y resources (but
   not for the command to finish, or even start running).
7. Go back to 1).
