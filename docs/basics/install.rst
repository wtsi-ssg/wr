Install
=======

coming soon...



The download .zip should contain the wr executable, this README.md, and a
CHANGELOG.md. 
You can use the wr executable directly from where you extracted it, or move it
to where you normally install software to.


The wr executable must be available at that same absolute path on all compute
nodes in your cluster, so you need to place it on a shared disk or install it in
the same place on all machines. In cloud environments, the wr executable is
copied for you to new cloud servers, so it doesn't need to be part of your OS
images. If you use config files, these must also be readable by all nodes (when
you don't have a shared disk, it's best to configure using environment
variables). If you are ssh tunnelling to the node where you are running wr and
wish to use the web interface, you will have to forward the host and port that
it tells you the web interface can be reached on, and/or perhaps also dynamic
forward using something like nc. An example .ssh/config is at the end of this
document.


Example .ssh/config

If you're having difficulty accessing the web frontend via an ssh tunnel, the
following example ~/.ssh/config file may help. (In this example, 11302 is the
web interface port that wr tells you about.)

.. code-block:: console
    :name: ssh-forwarding-example

    Host ssh.myserver.org
    LocalForward 11302 login.internal.myserver.org:11302
    DynamicForward 20002
    ProxyCommand none
    Host *.internal.myserver.org
    User myusername
    ProxyCommand nc -X 5 -x localhost:20002 %h %p

You'll then be able to access the website at
https://login.internal.myserver.org:11302 or perhaps https://localhost:11302
