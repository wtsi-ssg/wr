Problems
========

If you run in to problems with wr itself, the first thing to do is check your
log file. By default this will be ``~/.wr_production/log`` (or log.openstack for
a cloud deployment, available after tearing down).

If the logs don't say much or indicate any problem, try starting the manager
in ``--debug`` mode and replicate the problem.

If there's still nothing in the logs, you may also have a problem with logging,
so try starting the manager in foreground mode (``-f``), and keep that terminal
open (where log messages will appear) while you open another terminal to
replicate the problem.

Once you have some error messages, if the message doesn't make the solution
obvious, please seek help from the developers by talking to us on
`gitter <https://gitter.im/wtsi-wr>`_, or by creating an issue on
`github <https://github.com/VertebrateResequencing/wr/issues/new>`_.
