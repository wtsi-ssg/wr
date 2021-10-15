Nextflow
========

https://www.nextflow.io is a workflow runner like wr. You might want to use it
because of its easy to write and powerful DSL which can describe workflows in a
sharable format. wr, on the other hand, has no native workflow description
language; instead you figure out what commands you want to run as part of your
workflow yourself, and add them to wr.

While nextflow has comprehensive support for a wide variety of platforms on
which to execute your workflows, there is room for improvement in 2 areas:

1. When using an LSF cluster, Nextflow does not create job arrays, nor does it
   re-use job slots. This means on a contended cluster, there may be efficiency
   issues, and your jobs may take time to get scheduled. Each time one of the
   processes in your workflow completes, you may have to wait again before the
   next one is scheduled.
2. Native OpenStack support has not been implemented. The Kubernetes support can
   be used instead, but this does not support S3, and so you must also set up
   shared storage for Nextflow state. Furthermore, there are no satisfactory
   solutions for easily deploying a kubernetes cluster in OpenStack that can
   auto scale up and down, meaning resources may be wasted while your workflow
   does little or when it has completed.

wr has excellent LSF and OpenStack support, solving the above issues.

Thus, it is desirable to have Nextflow use wr as an execution backend to run
your Nextflow workflows in an LSF cluster or in an OpenStack environment (which
has an S3-compatible object store available).

Support for wr in Nextflow is via the nf-wr plugin.

.. note::
    The plugin is not yet compatible with the latest versions of Nextflow. You
    need to use an older version, eg. ``NXF_VER=20.10.0 nextflow [...]``.

You can build it by doing::

    git clone https://github.com/nextflow-io/nf-wr.git
    cd nf-wr
    make assemble
    export NXF_CLASSPATH=$PWD/build/libs/nf-wr-1.0.0.jar

You can then use Nextflow as normal, installed in its usual way.

.. note::
    You'll need the Java 8 JDK to build Nextflow, and the Java 8 JRE to run
    Nextflow.

Getting Started
---------------

For Nextflow to use wr, it must be able to communicate with wr, which means
knowing what machine wr is running on, what port it is listening to, and what
security token and certificate to use.

If you start (``wr manager start``) or deploy (``wr cloud deploy``) wr from the
same machine that you will run Nextflow, and use wr with it's default settings,
Nextflow will be able to guess all the details, so you just need to configure
Nextflow to use wr, by adding this to your ```nextflow.config``::

    executor {
        name='wr'
    }

If you're running wr in the 'development' deployment, change the above to::

    executor {
        name='wr'
        wr {
            deployment='development'
        }
    }

If you've configured wr to use a specific non-default managerweb (eg. 46408), or
a non-default managerdir (eg. ``/shared/disk/.wr``), (or non-default
managertokenfile or managercafile) then change the above to::

    executor {
        name='wr'
        wr {
            endpoint='https://localhost:46408'
            tokenpath='/shared/disk/.wr_production/client.token'
            cacertpath='/shared/disk/.wr_production/ca.pem'
        }
    }

.. note::
    Both wr and Nextflow, by default, use some disk space in your home
    directory. If that is full, you can configure wr's managerdir to be
    somewhere else (ideally an ssd, avoiding lustre if performance is a concern,
    though NFS is fine), and change the capsule cache directory used by Nextflow
    with ``export CAPSULE_CACHE_DIR=/a/disk/with/enough/space``.

Using it with LSF
-----------------

From your LSF head node, start wr in LSF mode::

    wr manager start -s lsf

Since Nextflow itself has quite significant memory requirements, you may need to
run Nextflow via LSF. Since that means Nextflow will run on some unknown machine
and not the machine that wr is running on, you must configure the endpoint
correctly. When wr starts it tells you that it started on a certain IP:port, and
where the web interface can be reached, which features a port in the address.
The endpoint to set will be the IP (or hostname of your LSF head node, if you
know that) and the web interface port.

For example, if ``wr manager start -s lsf`` tells you::

    > INFO[07-04|09:50:44] wr manager v0.18.1-37-g16f06e0 started on 172.17.27.150:46407, pid 23469

    > INFO[07-04|09:50:44] wr's web interface can be reached at https://localhost:46408/?token=QBM9zH0bNhV6OZdreKi1BI5DTq72kdWN0Vgaw3bvzF0

Then your nextflow.config should contain::

    executor {
        name='wr'
        wr {
            endpoint='https://172.17.27.150:46408'
        }
    }

Now you can submit your nextflow job as normal, eg::

    bsub -o run.o -e run.e -q yesterday -M 8000 -R 'select[mem>8000] rusage[mem=8000]' "./nextflow workflow.nf"
    tail -f run.o

Using it with OpenStack
-----------------------

From your local machine, :doc:`deploy wr to OpenStack </schedulers/openstack>`.
If you don't have your OpenStack image set up to mount a shared disk, you'll
also need :doc:`a working s3 setup </advanced/s3>`. The rest of this guide
assumes the S3 case, but you can ignore the S3-related advice if using a shared
disk.

If you will be using Docker or Singularity containers, or your workflow relies
on any other software to be installed, you will also need to to tell wr to use
an image you have created that has this software installed, or tell wr to run a
script that installs the software on some standard image at boot up time. Make
sure that Docker's default network does not interfere with the network that wr
will create or any other needed network.

If you want to use Singularity containers, this is more complicated than Docker
since the images must exist at the same absolute local path on the machine you
run Nextflow from, and the machine where the process actually runs. wr will
autoscale by creating new instances within OpenStack to run processes as
necessary, so while Nextflow may download an image locally, it will not be
available on any newly created instance, and processes will fail. One way around
this is to pre-download all your required images and store them in S3. Then use
a script with wr that mounts this S3 location, eg. ``mount.sh``::

    sudo apt-get update
    sudo apt-get install -y build-essential git libfuse-dev libcurl4-openssl-dev libxml2-dev mime-support automake libtool pkg-config libssl-dev git
    git clone https://github.com/s3fs-fuse/s3fs-fuse
    cd s3fs-fuse/
    ./autogen.sh
    ./configure --with-openssl
    make
    sudo make install
    mkdir /home/ubuntu/singularity_cache
    s3fs -o url=https://cog.sanger.ac.uk -o endpoint=us-east-1 -o sigv2,noatime,rw,uid=1000,gid=1000,umask=0002,allow_other mysingularitybucket /home/ubuntu/singularity_cache

.. note::
    In the future, ``wr cloud deploy`` may have an option to mount a bucket for
    you, making this much easier. Get in touch if you'd like this feature sooner
    rather than later.

Deploy using your desired image and/or script, and mention any config files your
script might need (``~/.s3cfg`` is copied over by default, but if following the
above example, we also need ``~/.passwd-s3fs`` for s3fs)::

    source ~/my_openstack.rc
    wr cloud deploy -o ubuntu-with-my-software -s mount.sh --config_files '~/.s3cfg,~/.passwd-s3fs'

Now wr will create instances within OpenStack that run your image and mount your
singularity bucket.

The next step is to configure Nextflow with your S3 details, and enable docker
or singularity if desired. Following the above example where we mount a
singularity bucket, ``nextflow.config`` would look like (in addition to the
executor block for wr)::

    docker.enabled = false

    singularity {
    enabled     = true
    autoMounts  = false
    cacheDir = '/home/ubuntu/singularity_cache'
    }

    aws {
        accessKey = 'MYACCESSKEY'
        secretKey = 'mysecret'
        client {
        endpoint = 'https://cog.sanger.ac.uk'
        signerOverride = "S3SignerType"
        }
    }

Your workflow should specify inputs and outputs as being in S3. An example
workflow.nf being::

    #!/usr/bin/env nextflow

    Channel.fromPath('s3://bucket/inputs/*.input').set { inputs_ch }

    process capitalize {
        input:
        file x from inputs_ch
        output:
        file 'file.output' into outputs_ch
        script:
        """
        cat $x | tr [a-z] [A-Z] > file.output
        """
    }

    outputs_ch
        .collectFile()
        .println{ it.text }

Finally, run Nextflow from the same machine that you did the deploy from, being
sure to specify that your working directory is in S3::

    ./nextflow workflow.nf -w s3://bucket/nextflow/work

If following this Singularity example where the cachDir is specified as
``/home/ubuntu/singularity_cache``, this will fail if your local machine does
not have that directory (eg. because it is not an Ubuntu machine). Instead you
can ssh to the instance that wr first creates during the deploy (it prints out
instructions on how to do this ssh), and run nextflow directly within OpenStack.
If doing this, be sure to set the endpoint in your ``nextflow.config`` back to
localhost, eg. ``endpoint='https://localhost:46408'``.

Because wr will create the smallest instances possible to run your workflow
processes, and also run processes on the first instance where wr (and perhaps
Nextflow itself) is running, it's important that your workflow specifies how
much CPU, RAM and disk each process uses. Otherwise you could end up filling the
first instance and killing wr, Nextflow or the whole instance's Operating
System.

You can avoid this possibility completely by adding the ``--max_local_ram 0``
option to your ``wr cloud deploy`` command. This will prevent any workflow
commands running on the same instance as wr. But your processes themselves may
still fail if they try to use more RAM or disk than the instances they are run
on have. So do take the time to add conservative resource usage specifications
to your workflows. Consider adding 10 more GB of disk space than you think your
process needs, since the Operating System itself will use some of the space.

Once your workflow has completed, you can use something like ``s3cmd ls -r
s3://bucket/nextflow/work`` to see all your files. It is not recommended to use
``publishDir`` in your workflow if at all possible, because S3 does not support
symlinks, and so a copy will be forced, which both takes time and doubles your
S3 quota usage for your final files.

When you've competed all your workflows, you can clean up by running ``wr cloud
teardown``.

Servers can go "bad"
--------------------

When executing your workflow, wr may create new OpenStack servers on which to
run your Nextflow processes. However it is possible for these servers to go
"bad". Going bad means they can no longer be ssh'd to. This could be due to a
temporary networking issue, or it could be because the server has crashed.

Because the problem might only be temporary, wr initially only tells you about
the issue (on its status webpage), but lets the servers continue to exist and
assumes processes are still running on them.

If you did nothing, you could end up with lots of bad servers that can't run any
processes, while wr thinks it is running all those processes, and so you may
find nothing is actually running anymore.

There is, however, an ``--auto_confirm_dead`` option that defaults to 30 mins,
which will destroy "bad" servers that remain bad for 30 mins, freeing up
resources and letting wr create new healthy servers on which to run your
processes. If you notice this happening a lot, you may wish to increase the
number of minutes to allow yourself more time to investigate why your servers
keep going bad. (It will likely be due to one of your nextflow processes using
too much memory or disk space.)
