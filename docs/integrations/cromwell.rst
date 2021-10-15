Cromwell
========

`Cromwell <https://cromwell.readthedocs.io/en/stable/>`_ is a workflow runner
like wr. You might want to use it because of its support for WDL which can
describe workflows in a sharable format. wr, on the other hand, has no native
workflow description language; instead you figure out what commands you want to
run as part of your workflow yourself, and add them to wr.

While Cromwell has comprehensive support for a wide variety of platforms on
which to execute your workflows, there is room for improvement in 2 areas:

1. When using an LSF cluster, Cromwell does not create job arrays, nor does it
   re-use job slots. This means on a contended cluster, there may be efficiency
   issues, and your jobs may take time to get scheduled. Each time one of the
   processes in your workflow completes, you may have to wait again before the
   next one is scheduled.
2. OpenStack support has not been implemented.

wr has excellent LSF and OpenStack support, solving the above issues.

Thus, it is desirable to have Cromwell use wr as an execution backend to run
your WDL workflows in an LSF cluster or in an OpenStack environment.

Cromwell config
---------------

Support for wr in Cromwell can be implemented simply by changing Cromwell's
config so that wr is called. An example cromwell.conf::


    include required(classpath("application"))
    backend {
    default = Local
    providers {  
        Local {
        actor-factory = "cromwell.backend.impl.sfs.config.ConfigBackendLifecycleActorFactory"
        config {
            run-in-background = true

            # The list of possible runtime custom attributes.
            runtime-attributes = """
            String? docker
            String? docker_user
            String? wr_cwd
            String? wr_cloud_script
            String? wr_cloud_os
            String? wr_cloud_flavor
            """

            # Submit string when there is no "docker" runtime attribute.
            submit = """
            echo "/usr/bin/env bash ${script}" | wr add \
                --cwd ${wr_cwd} \
                --cwd_matters \
                --on_failure '[{"run":"write_cromwell_rc_file.sh"},{"remove":true}]'
                --cloud_script ${wr_cloud_script} \
                --cloud_os ${wr_cloud_os} \
                --cloud_flavor ${wr_cloud_flavor} \
                --deployment development
            """
            
            root = "<s3_mount_path>/cromwell-executions"

            # File system configuration.
            filesystems {
            local {
                localization: [
                "hard-link", "soft-link", "copy"
                ]
                caching {
                # When copying a cached result, what type of file duplication should occur. Attempted in the order listed below:
                duplication-strategy: [
                    "hard-link", "soft-link", "copy"
                ]
                hashing-strategy: "file"
                check-sibling-md5: false
                }
            }
            }

            default-runtime-attributes {
            failOnStderr: false
            continueOnReturnCode: 0
            }
        }
        }
    }
    }

Where ``write_cromwell_rc_file.sh`` is an executable bash script in your PATH
that will create the rc files that Cromwell needs in the case that a wr runner
is forcibly killed before Cromwell can create its rc file::

    #!/bin/bash
    export cwd=$PWD
    echo search rc file in cwd: $cwd
    attempt_dir=$(ls -d $cwd/* | grep attempt- | tail -n 1)
    echo attempt_dir: $attempt_dir
    if [ ! -z "$attempt_dir" ]
    then
        export cwd=$attempt_dir
    fi
    if test -f $cwd/execution/rc; then
        echo rc file already present at ${cwd}/execution/rc
        cat ${cwd}/execution/rc
    else
        echo write rc file to $cwd/execution/rc
        echo 1 > $cwd/execution/rc
        echo 1 > $cwd/execution/wr.failure
    fi

There may be other ways of configuring things, eg. hardcoding values without the
need to pass options in from your WDL files. Ideally you should also pass the
``-i`` option to ``wr add``, with a value that is unique to a particular
execution of a workflow, so that you can use ``wr status -i <unique value>``
later on to get status and stats on that execution.
