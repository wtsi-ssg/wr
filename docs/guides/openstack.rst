OpenStack
=========

For an example of using wr to do some real work in an OpenStack+S3 environment,
this walkthrough shows how you would carry out the "Worked Example" given at the
bottom of http://www.htslib.org/workflow.

Prerequisites: A linux machine that has API access to an OpenStack environment
and that can also reach an S3-like object store which is also accessible within
that OpenStack environment. s3cmd (or some other tool capable of creating
buckets) also needs to be installed on your linux machine.

Install wr
----------

First install the latest version of wr on your local linux machine::

    mkdir wr
    cd wr
    curl -s -L https://github.com/VertebrateResequencing/wr/releases/latest | egrep -o '/VertebrateResequencing/wr/releases/download/v[0-9\.]*/wr-linux-x86-64.zip' | head -n 1 | wget --base=http://github.com/ -q -i - -O wr.zip
    unzip wr.zip

Configuration
-------------

S3
^^

Create a standard ``~/.s3cfg`` file with your S3 credentials in it, if you don't
already have one. It might look like::

    [default]
    access_key = YOURACCESSKEY
    secret_key = yoursecretkey
    encrypt = False
    host_base = cog.sanger.ac.uk
    host_bucket = %(bucket)s.cog.sanger.ac.uk
    use_https = True

.. note::
    Ask your S3 administrator for the correct values to fill in here. (The
    host_* values in the example above are only suitable for use within the
    Sanger Institute).

OpenStack
^^^^^^^^^

You will need the set of OS_* environment variables defined; these provide the
credentials for accessing your OpenStack system. If you don't already have these
defined, ask your OpenStack administrator for details on how to access your
"Horizon" web interface to OpenStack, for which you'll need a user name and
password.

Once logged in, go to the "Project" tab, then the "API Access" tab. From there
click the "Download OpenStack RC File" button. If given a choice, pick the
highest version Identity API file. Transfer this file to your linux machine if
necessary and source it. This might look like::

    source ~/[projectname]-openrc.sh

It might ask you to enter the same password you used to log in to the web
interface.

Having sourced the file, the necessary OpenStack environment variables will be
available to wr when it needs them later.

wr
^^

wr's default configuration values may be good enough, but you should ask your
OpenStack/ system administrator if there are any DNS IPs you should use in
particular, or if you should only use certain OpenStack "flavors". You also need
to be aware of the names of images that are available in your OpenStack
installation, and of the username needed to log in to them.

Since all of these installation-specific things typically don't change, it is
convenient to set the appropriate values in wr's config file.

Start by saving a default config file to where wr will find it::

    wr conf --default > ~/.wr_config.yml

Then look through it and edit any values as necessary. For example, you might
change these ones::

    cloudflavor: "^[mso].*$"
    cloudflavorsets: "s2;m2;m1;o2"
    clouddns: "172.18.255.1,172.18.255.2,172.18.255.3"
    cloudos: "bionic-server"
    clouduser: "ubuntu"
    cloudram: 2048

.. note::
    These example settings are suitable for use at the Sanger Institute, but
    likely not anywhere else.

Now when wr creates OpenStack instances for you, they will by default be running
your Ubuntu bionic-server image, have at least 2GB of ram, only use flavours
that start with one of the letters 'm', 's', or 'o', and use the given IP
addresses to resolve DNS requests.

Prepare S3
----------

If you don't already have a "bucket" (root folder) to store your data in S3, use
a tool like s3cmd to create one, eg.::

    s3cmd mb s3://mybucket

Note that bucket names need to be globally unique and its best if you stick to
just letters and numbers (avoid any punctuation characters).

You could at this point use s3cmd again to upload the input data needed for the
workflow later, but you should test upload performance from your local linux
machine to S3 compared to from an OpenStack instance to S3.

Since it could be faster to upload data from within OpenStack (it is at the
Sanger Institute), we'll do our input data uploads using wr later on.

Deploy wr to the cloud
----------------------

::

    ./wr cloud deploy

This command will create all the necessary cloud resourcess in OpenStack for
you, including a network, security group, security key and 1 initial server, on
which wr is installed and wr's queue manager is started. It gives you a URL you
can visit in your browser to see wr's status web interface. It also gives you
the IP address of the OpenStack instance it created.

Figure out how to install your desired software
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

Before we can run our analysis, we will need to install our analysis software
on the OpenStack instances that wr brings up. In our case, we need samtools and
bwa.

First ssh to the instance that wr created for us, using the ip address it told
us about::

    ssh -i ~/.wr_production/cloud_resources.openstack.key ubuntu@[ipaddress]

Now check to see if our default image happens to have our software already
installed::

    which samtools
    which bwa

If these don't return paths to the executables, you'll have to figure out how to
install the software, and make note of what you did. For samtools and bwa, these
commands worked on the image I was using::

    sudo apt-get update
    sudo apt-get install -y gcc make autoconf zlib1g-dev libbz2-dev liblzma-dev libncurses5-dev
    wget "https://github.com/samtools/samtools/releases/download/1.4/samtools-1.4.tar.bz2"
    tar -xvjf samtools-1.4.tar.bz2
    rm samtools-1.4.tar.bz2
    cd samtools-1.4
    ./configure
    make
    sudo make install
    cd
    rm -fr samtools-1.4

    wget "https://downloads.sourceforge.net/project/bio-bwa/bwa-0.7.15.tar.bz2?r=https%3A%2F%2Fsourceforge.net%2Fprojects%2Fbio-bwa%2Ffiles%2F&ts=1492592278&use_mirror=netcologne" -O bwa-0.7.15.tar.bz2
    tar -xvjf bwa-0.7.15.tar.bz2
    cd bwa-0.7.15
    make
    sudo mv bwa /usr/local/bin/
    cd
    rm -fr bwa*

Record all these commands in a file on your local linux machine, eg. by putting
them in a text file called ``samtools_bwa_install.sh``.

.. tip::
    Alternatively, if it is a time consuming install process, you might use
    Horizon to create a new image based on this OpenStack instance, that you'd
    specify later on during ``wr add`` with the ``--cloud_os`` option instead of
    using the ``--cloud_script`` option this walkthrough will talk about.)

    Or you could see if your desired software was available in a container and
    use an image with eg. docker correctly installed and configured, and alter
    the subsequent commands in this tutorial as appropriate for running your
    software from within docker.

Exit from the OpenStack server::

    exit

Now, just to demonstrate that this will work when you haven't ssh'd to an
OpenStack instance to manually install software, destroy all the OpenStack
resources previously created by the deploy::

    ./wr cloud teardown

And then deploy again to get to a fresh state where neither samtools nor bwa
have been installed::

    ./wr cloud deploy

Add your commands
-----------------

At it's heart, wr is a job (command) queue. You add jobs to its queue, and then
wr does whatever is necessary to run your jobs on the available compute
resources. In our case, having done a cloud deployment, that means wr will spawn
new OpenStack instances (picking the cheapest flavor capable of running the
command) to run the commands we queue up, and then when the commands complete wr
will terminate those instances (known as auto scaling up and down).

In this walkthrough we have some samtools and bwa commands we want to run
against some input data. The input data along with some of the intermediate
files produced along the way to the final output files could be useful in the
future, so we'd want to keep them. Because of this we can think of our work as
being formed of multiple steps:

1. Store one of the input fastq files in S3
2. Store the other input fastq file in S3
3. Store reference fasta file in S3
4. Produce a samtools reference index and store in S3
5. Produce a bwa reference index and store in S3
6. Align the pair of fastqs from steps 1&2 with bwa mem (using the index from
   step 5), sort with samtools, convert to cram with samtools (using the index
   from step 4), store the results in S3

Steps 1, 2 and 3 are independent. Steps 4 and 5 can only proceed once step 3 has
completed. Step 6 can only proceed once steps 1, 2, 3, 4 and 5 have completed.

To achieve this workflow we will make use of wr's dependency features, along
with its built-in S3 mounting capability.

In production it would be better to generate the more complicated wr add JSON
and specify all your commands in one go, but for this walkthrough we'll break
this out in to 6 separate calls to ``wr add`` so we can use the simpler command
line options. You do not have to wait for one step to complete before adding the
command for the next; just do all 6 adds in quick succession and watch the
progress on the status web page (or use ``./wr status``).

Add the command for step 1 (the curl download is wrapped in a perl system call
since our use of head for demo purposes actually results in the pipe from curl
through to head breaking, which wr would regard as a failure)::

    echo "perl -e 'system(q[curl -sS ftp://ftp.sra.ebi.ac.uk/vol1/fastq/SRR507/SRR507778/SRR507778_1.fastq.gz | gzip -d | head -100000 > SRR507778_1.fastq]) && die'" | ./wr add -i uploads -g curl -e SRR507778.fastqs.upload --mounts 'uw:mybucket/fastq/SRR507/SRR507778'

To explain the options given to `./wr add`:

* ``-i uploads`` is purely for display purposes: this command will show on the
  status webpage under the 'uploads' heading.
* ``-g curl`` lets us specify a resource requirements group on which wr's
  resource usage learning will act. The next time we add any command with this
  same ``-g``, wr will take in to account the actual resources used when it ran
  previous commands with that ``-g``. We choose a name that we think we could
  use again on future commands, and that is specific enough that those future
  commands are likely to have the same resource requirements.
* ``-e SRR507778.fastqs.upload`` lets us specify a dependency group. Other
  commands (such as our step 6 command, see later) will be able to refer to
  this group name to become dependent on commands with this group having
  completed.
* ``--mounts`` gives us a convenient way of specifying a particular S3 remote
  "sub-directory" (S3 is an object store and doesn't really have directories,
  but we can pretend it is like a normal POSIX filesystem with wr) as the
  working directory the curl command will run in. The first character 'u' means
  'uncached', the second character 'w' means 'writeable', and the desired bucket
  and sub-directory comes after a colon. In this example we try to emulate the
  directory structure of the ftp site we're copying the fastq file from; note
  that these "directories" don't have to actually exist in our S3 bucket
  beforehand.

Similarly, add the command for step 2::

    echo "perl -e 'system(q[curl -sS ftp://ftp.sra.ebi.ac.uk/vol1/fastq/SRR507/SRR507778/SRR507778_2.fastq.gz | gzip -d | head -100000 > SRR507778_2.fastq]) && die'" | ./wr add -i uploads -g curl -e SRR507778.fastqs.upload --mounts 'uw:mybucket/fastq/SRR507/SRR507778'

Add the command for step 3, changing the remote "sub-directory" and -e option as appropriate::

    echo "curl -sS ftp://ftp.ensembl.org/pub/current_fasta/saccharomyces_cerevisiae/dna/Saccharomyces_cerevisiae.R64-1-1.dna_sm.toplevel.fa.gz | gzip -d > ref.fasta" | ./wr add -i uploads -g curl -e yeast.ref.upload --mounts 'uw:mybucket/refs/saccharomyces_cerevisiae'

Add the command for step 4::

    echo "samtools faidx ref.fasta" | ./wr add -m 4G -i indexing -g samtools.faidx.yeast -e samtools.faidx.yeast -d yeast.ref.upload --mounts 'uw:mybucket/refs/saccharomyces_cerevisiae' --cloud_script samtools_bwa_install.sh

* ``-m 4G`` is our way of specifying that we think `samtools faidx` might need
  4GB of memory to run. (If wr learns and knows better it will ignore this.)
* ``-g samtools.faidx.yeast`` is so specific because we imagine that running
  faidx on a yeast-sized genome might take different amounts of time and memory
  than running it on other species.
* ``-e samtools.faidx.yeast`` will allow us to depend on our yeast reference
  having been indexed both in step 6 and in the future should we add any
  commands that also need this index file.
* ``-d yeast.ref.upload`` is how we specify that this command should wait until
  step 3 completes.
* ``--mounts`` specifies the place we uploaded the reference fasta to in step 3.
* ``--cloud_script samtools_bwa_install.sh`` will result in this command only
  running on an OpenStack server that had this script run when it booted up; wr
  will create a new server and run the script on it if necessary.

Similarly, add the command for step 5::

    echo "bwa index ref.fasta" | ./wr add -m 4G -i indexing -g bwa.index.yeast -e bwa.index.yeast -d yeast.ref.upload --mounts 'uw:mybucket/refs/saccharomyces_cerevisiae' --cloud_script samtools_bwa_install.sh

Finally, add the command for step 6::

    echo "bwa mem ref.fasta SRR507778_1.fastq SRR507778_2.fastq | samtools sort -O bam -l 0 -T /tmp - | samtools view -T ref.fasta -C -o SRR507778.cram -" | ./wr add -m 4G -i step6 -g bwatocram.yeast -d "SRR507778.fastqs.upload,samtools.faidx.yeast,bwa.index.yeast" --mounts 'ur:mybucket/fastq/SRR507/SRR507778,cr:mybucket/refs/saccharomyces_cerevisiae,uw:mybucket/crams/SRR507' --cloud_script samtools_bwa_install.sh

* We don't bother setting an ``-e`` option since no commands in our workflow
  depend on this. Though, perhaps we ought to plan for the future and imagine
  that we might want to add commands that do depend on this, in which case we
  should add an ``-e`` option after all.
* ``-d "SRR507778.fastqs.upload,samtools.faidx.yeast,bwa.index.yeast"`` is how
  we specify that this command should wait until steps 1, 2, 4 and 5 complete
  (and by implication, step 3).
* The ``--mounts`` option here multiplexes 3 of our S3 bucket "directories"
  together, so that our command sees the contents of all them in its working
  directory. The first is our fastqs directory, which is read-only. The second
  is our reference directory, also read-only and this time cached because we
  know our command will read the indexes and reference files multiple times. The
  third is our separate output directory, where we specify 'w' for writeable.
  Any writes that our command carries out will end up in this S3 directory.
  (Reads could come from any of the directories, preferring those paths
  specified earliest.)

You'll note that (depending on the speed of spawning another OpenStack instance)
steps 1, 2 and 3 run simultaneously while steps 4, 5 and 6 wait in 'dependent'
state until their dependencies complete, before they too start to run. If you
see commands 'pending' they are waiting for new OpenStack servers to be created.

wr will learn how much memory step 6 actually takes, so the next time you add a
command with ``-g bwatocram.yeast -m 4G``, it will consider the real value
instead of 4GB (unless you also set ``-o 2``), and potentially run the command
on a cheaper flavor.

Once complete, if you don't plan on doing any more work soon, teardown so you're
not wasting resources::

    ./wr cloud teardown

Confirm that your result cram is available::

    s3cmd ls s3://mybucket/crams/SRR507/