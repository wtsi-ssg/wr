Kubernetes
==========

wr has the ability to self-deploy the manager to any Kubernetes cluster, which
might be running locally, in OpenStack, or in public clouds like AWS and GCP. It
has been tested with kubernetes versions 1.9.5 and 1.10.

.. note::
    The kubernetes support has not been well tested and may be out of date; if
    you have a need for this to work, please :ref:`get_in_touch`.

There are myriad methods of starting a Kubernetes cluster, see
`kubernetes.io <https://kubernetes.io/docs/setup/pick-right-solution/>`_ for an
overview and details.

As long as ``kubectl get nodes`` returns some 'Ready', 'node' role nodes, ``wr
k8s deploy`` should also work. After that, use wr normally, and ``wr k8s
teardown`` when you're done.