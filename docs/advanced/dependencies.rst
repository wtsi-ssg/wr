Dependencies
============

To construct a workflow with multiple steps, where later steps only run after
earlier steps have completed, you must either arrange to only add the commands
for later steps after earlier ones have completed, or add all the commands for
all steps without waiting, but specify dependencies.

When adding jobs to the manager's queue with ``wr add``, you can specify
dependencies using:

``--dep_grps``
    A comma-separated list of arbitrary names. Apply this to commands that other
    commands will depend upon.

``--deps``
    A comma-separated list of the dep_grp of other commands that this command
    should depend on. Dependencies specified in this way are 'live', causing
    this command to be automatically re-run if any commands with any of the
    dep_grps it is dependent upon get added to the queue.

    .. note::
        You can also specify "static" dependencies when using the JSON input to
        ``wr add``, specifying "cmd_deps" as an array of JSON objects with "cmd"
        and "cwd" name:value pairs (if cwd doesn't matter for a cmd, provide it
        as an empty string). Once resolved they do not get re-evaluated.

When a command depends on others, it will only be scheduled for execution after
the other commands have completed successfully (exited 0).

For example, imagine you had a 2 step workflow designed to do the job of
'xyzing' your data, that first ran an executable ``foo`` on a particular input
file, then ran another command ``bar`` using foo's output as its input. 

If you then had input files '1.txt', '2.txt' and '3.txt', you could add the
commands for this workflow like::

    echo "foo 1.txt > 1.out.foo" | wr add --dep_grps 'input1' --cwd_matters 
    echo "foo 2.txt > 2.out.foo" | wr add --dep_grps 'input2' --cwd_matters
    echo "foo 3.txt > 3.out.foo" | wr add --dep_grps 'input3' --cwd_matters
    echo "bar 1.out.foo > 1.out.bar" | wr add --deps 'input1' --cwd_matters
    echo "bar 2.out.foo > 2.out.bar" | wr add --deps 'input2' --cwd_matters
    echo "bar 3.out.foo > 3.out.bar" | wr add --deps 'input3' --cwd_matters

.. note::
    In reality, it would be more efficient to add all these commands in one go,
    using JSON to specify the dependencies, and you should also specify rep_grp
    along with :doc:`other details </basics/add>`.

    You should also be careful to be as specific and meaningful as possible in
    your dep_grp names, so you won't re-use the same dep_grp in future for
    something that's actually different and unrelated.

While this will work for this single 2 step workflow, it isn't flexible for
possible integration with other workflows and extending the workflow in future.

Instead, you should do something like::

    echo "foo 1.txt > 1.out.foo" | wr add --dep_grps 'xyz,xyz.step1,xyz.step1.input1' --cwd_matters 
    echo "foo 2.txt > 2.out.foo" | wr add --dep_grps 'xyz,xyz.step1,xyz.step1.input2' --cwd_matters
    echo "foo 3.txt > 3.out.foo" | wr add --dep_grps 'xyz,xyz.step1,xyz.step1.input3' --cwd_matters
    echo "bar 1.out.foo > 1.out.bar" | wr add --dep_grps 'xyz,xyz.step2,xyz.step2.input1' --deps 'xyz.step1.input1' --cwd_matters
    echo "bar 2.out.foo > 2.out.bar" | wr add --dep_grps 'xyz,xyz.step2,xyz.step2.input1' --deps 'xyz.step1.input2' --cwd_matters
    echo "bar 3.out.foo > 3.out.bar" | wr add --dep_grps 'xyz,xyz.step2,xyz.step2.input1' --deps 'xyz.step1.input3' --cwd_matters

Now you have the option of adding a command that is:

* Dependent on the whole xyz workflow having completed,