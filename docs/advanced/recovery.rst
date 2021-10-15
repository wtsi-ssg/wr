Disaster Recovery
=================

The wr manager maintains the state of all your jobs in its on-disk database. In
the event of a disaster, such as the manager's process crashing, or the host
where the manager is running losing power, complete recovery of state is
possible when you start up the manager again if:

* The manager's database (or its database backup) is still accessible.
* The client token file is still accessible.
* The manager is started on a host with the same IP address as before (or you're
  using a domain, and you update the domain's IP to point to the server you
  start the manager on).
* You restart the manager within 24hrs of the disaster.

The last 3 points are only relevant if jobs were still running when the disaster
happened, and the disaster did not affect the runners.

In practice, this means that for the non-cloud schedulers (eg. local and LSF),
you don't have to do anything special, other than always bring up your manager
on the same machine, and make sure your disk doesn't die. To deal with the
possibility that your disk does die, configure wr's database backup to go to a
different disk than the primary copy.

For cloud schedulers (eg. OpenStack), you should:

* Configure database backups to go to an S3 bucket.
* Configure :doc:`your own security certificate and domain </advanced/security>`.
* Use something like infoblox to have your domain point to the IP address of the
  manager's host.
* Start the manager with the `--use_cert_domain` option (or configure that).
* Store your client token somewhere, and recreate the token file in the event of
  a recovery situation.

The easiest way of handling the last 3 points is to use ``wr cloud deploy
--set_domain_ip``. If the node running the manager gets destroyed, just deploy
again: any runner nodes will reconnect to the new manager.

If you bring up your own OpenStack node on which you ``wr manager start``,
you'll have to handle saving and restoring the token file yourself. The old
token file is needed so that the new manager uses the old token, allowing old
runners (which are using that old token) to reconnect to the new manager.

Ideal case
----------

If you satisfy all the above conditions, then when you start the manager again
following a disaster, it will be almost as if no disaster had occurred:

* Jobs that are still running will continue to run and complete as normal.
* Jobs that finished running while the manager was down will have a runner that
  stays up until the manager is started, so that the job's state can be stored.
* Jobs also affected by the disaster will initially come up in the new manager
  as "running", but after a short while the manager will realize something is
  wrong and convert these to "lost contact" state, which you can then confirm as
  dead to bury and then retry as desired.

.. note::
    It's possible for the manager to automatically determine that the lost jobs
    are really dead, if it can now ssh to the hosts where they ran. If the job's
    PIDs no longer exist, the jobs will be buried. If the jobs were added with
    multiple retries allowed, and there are retries remaining, they will instead
    be automatically rescheduled.
    
    So potentially you won't have to take any manual action following a disaster
    other than start the manager again.

Old clients can't connect to new manager
----------------------------------------

If you satisfy the very first point on this page, but not all the others, it
means that your jobs will be intact and correct, but you may end up restarting
some jobs from scratch:

* Jobs that are still running, and jobs that finished running while the manager
  was down, will not be able to contact the new manager to report their state,
  so the new manager will eventually put them in to "lost contact" state (along
  with jobs affected by the disaster).

To prevent yourself from running the same command more than once simultaneously,
you must manually go through your compute environment and ensure that there are
no wr runners still running, **before starting the manager again**. In local
mode this means you should look at all your processes and `kill -9` the process
groups of any runners. In LSF mode you should `bkill` all runners. In OpenStack
mode you should destroy all wr-created hosts.

Having killed off all old runners, it is then safe to start the manager, wait
for running jobs to become "lost contact", and then confirm those jobs are dead,
before finally retrying them if desired.
