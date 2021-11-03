Install
=======

The recommended way to install wr is to download a release from `github`_. Click
on "Assets" for the most recent release and download either the linux or macos
.zip file as appropriate. Windows is not supported.

.. _github: https://github.com/VertebrateResequencing/wr/releases

.. note::
    The macos version cannot currently be used to "deploy" to linux systems, so
    if you have a Mac but plan on using wr with a linux-based OpenStack system,
    download the linux version of wr and run it on a linux system, eg. one of
    your OpenStack nodes.

The downloaded .zip should contain the wr executable, a README.md, and a
CHANGELOG.md.

You can use the wr executable directly from where you extracted it, or move it
to where you normally install software to (ie. somewhere in your $PATH).

The wr executable must be available at that same absolute path on all compute
nodes in your cluster, so you need to place it on a shared disk or install it in
the same place on all machines.

.. note::
    In cloud environments, the wr executable is automatically copied for you to
    new cloud servers, so it doesn't need to be part of your OS images.

Build
-----

Alternatively to downloading the pre-built executable, you can build wr yourself
(check the go.mod file to see the minimum version of go required):

1. Install go on your machine according to https://golang.org/doc/install. An
   example way of setting up a personal Go installation in your home directory
   would be::

        export GOV=1.17.1
        wget https://dl.google.com/go/go$GOV.linux-amd64.tar.gz
        tar -xvzf go$GOV.linux-amd64.tar.gz && rm go$GOV.linux-amd64.tar.gz
        export PATH=$PATH:$HOME/go/bin

2. Download, compile, and install wr (not inside $GOPATH, if you set that)::

        git clone https://github.com/VertebrateResequencing/wr.git
        cd wr
        make

3. The ``wr`` executable should now be in ``$HOME/go/bin``.

.. note::
    If you don't have ``make`` installed and don't mind if ``wr version`` will
    not work, you can instead replace ``make`` above with::

        go install -tags netgo

Bash auto completion
--------------------

If you build wr yourself on the machine you will use wr on, and you use bash as
your shell, you can install bash auto-completion so you can tab complete the
various wr sub commands::

    wr completion bash > ~/.wr_autocomplete.sh
    source ~/.wr_autocomplete.sh
    echo "source ~/.wr_autocomplete.sh" >> ~/.bashrc
