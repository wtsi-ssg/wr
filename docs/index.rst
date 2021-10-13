wr: the workflow runner
=======================

.. meta::
    :description lang=en: wr: the high performance workflow runner.

`wr`_ is a workflow runner. You use it to run the commands in your workflow
easily, automatically, reliably, with repeatability, all while making optimal
use of your available computing resources.

wr is implemented as a polling-free in-memory job queue with an on-disk acid
transactional embedded database, written in go.

High performance
    Low latency and overhead, plus high performance at scale, lets you
    confidently run any number of any kind of command.

Real-time
    Real-time status updates with a view on all your workflows on one screen.

History
    Permanent searchable history of all the commands you ever ran, with details
    and summary statistics on how long they took to run and how much memory they
    used.

Dependencies
    "Live" dependencies allow for easy automation of on-going projects.
    Read more about :doc:`/advanced/dependencies`.

OpenStack
    Best-in-class OpenStack support, with increadibly easy deployment and
    auto-scaling up and down. All without you having to know anything about
    OpenStack.
    Read more about :doc:`/schedulers/openstack`.

S3
    Mount S3-like object stores, for an easy way to run commands against remote
    files whilst enjoying high performance.
    Read more about :doc:`/schedulers/s3`.


.. _wr: https://github.com/VertebrateResequencing/wr

First steps
-----------

* **Tutorial**: :doc:`Basics </tutorials/basic>`

* **Getting started**:
  :doc:`Install </basics/install>` |
  :doc:`Start the manager </basics/manager>` |
  :doc:`Add commands </basics/add>` |
  :doc:`Check status </basics/status>`

.. toctree::
   :maxdepth: 2
   :hidden:
   :caption: First steps

   /tutorial
   /basics/install
   /basics/manager
   /basics/add
   /basics/status

Advanced usage
--------------

Learn about some of the more advanced features of wr.

* **Dependencies**:
  :doc:`Specify dependencies </advanced/dependencies>`

.. toctree::
   :maxdepth: 2
   :hidden:
   :caption: Advanced usage

   /advanced/dependencies

How-to Guides
-------------

These guides will help walk you through specific use cases.

Support
-------

The best way to report bugs or make feature requests is to `create an issue on
github <https://github.com/VertebrateResequencing/wr/issues/new>`.

To chat with the developers and get help or discuss feature requests, join us on
`gitter <https://gitter.im/wtsi-wr>`.


If you'd like to contribute to wr's development, follow this guide.
