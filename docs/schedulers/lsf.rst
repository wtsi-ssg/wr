LSF
===

docs coming soon...

By default, this scheduler will use heursitics that look at the configuration of
your LSF queues and the resource requirements (cpu, time, memory) of your jobs
to determine the best LSF queue to submit jobs to. You can instead override this
by explicitely providing a ``--queue`` to ``wr add``.

You can also use the ``--misc`` option of ``wr add``, where the value you
specify will be passed directly to LSF. For example, ``--misc '-R avx'`` might
result in the manager running ``bsub -R avx [...]``. To avoid quoting issues,
surround the ``--misc`` value in single quotes and if necessary use double
quotes within the value; do NOT use single quotes within the value. Eg. ``--misc
'-R "foo bar"'``.
