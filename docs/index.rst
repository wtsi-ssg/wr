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
    Read more about :doc:`/advanced/s3`.

.. _wr: https://github.com/VertebrateResequencing/wr

First steps
-----------

* **Tutorial**: :doc:`Basics </guides/basic>` (Why use wr? What does it do?
  How do you use it?)

* **Getting started**:
  :doc:`Install </basics/install>` |
  :doc:`Start the manager </basics/manager>` (:doc:`resolve problems with that </basics/problems>`) |
  :doc:`Add jobs </basics/add>` |
  :doc:`Check status </basics/status>`

* **Manipulate jobs**:
  :doc:`Retry </basics/retry>` |
  :doc:`Kill </basics/kill>` |
  :doc:`Remove </basics/remove>` |
  :doc:`Modify </basics/mod>` |
  :doc:`Limit </basics/limit>`

.. toctree::
   :maxdepth: 2
   :hidden:
   :caption: First steps

   /guides/basic
   /basics/install
   /basics/manager
   /basics/problems
   /basics/add
   /basics/status
   /basics/retry
   /basics/kill
   /basics/remove
   /basics/mod
   /basics/limit

Schedulers
----------

After you've added jobs to wr's queue, wr schedules the jobs so they will be
executed on your compute resources efficiently. You can learn more about how it
schedules :doc:`in general </schedulers/schedulers>`, but you probably just need
to pick the right scheduler for your circumstances:

* :doc:`Local </schedulers/local>` for executing on just your local machine.
* :doc:`LSF </schedulers/lsf>` for executing on an LSF cluster.
* :doc:`OpenStack </schedulers/openstack>` for executing on an OpenStack cluser.

.. toctree::
   :maxdepth: 2
   :hidden:
   :caption: Schedulers

   /schedulers/schedulers
   /schedulers/local
   /schedulers/lsf
   /schedulers/openstack
   /schedulers/kubernetes

Advanced topics
---------------

Learn about some of the more advanced features of wr.

* :doc:`Create workflows using dependencies </advanced/dependencies>`
* :doc:`Work with files in S3 </advanced/s3>`
* :doc:`Improve security </advanced/security>`
* :doc:`Disaster recovery </advanced/recovery>`
* :doc:`Use wr via its REST API </advanced/rest>`

.. toctree::
   :maxdepth: 2
   :hidden:
   :caption: Advanced

   /advanced/dependencies
   /advanced/s3
   /advanced/security
   /advanced/recovery
   /advanced/rest
   /advanced/contribute

Integrations
------------

While wr can be used by itself to execute workflows, you need to manually figure
out a way to specify all the commands you want to run and their dependencies.

Other workflow management systems offer ways of easily specifying your workflows
in general and shareable ways, but might not be as efficient or capable as wr
in actually scheduling, executing and tracking the commands.

For these systems, wr can integrate with them for use as an execution "backend",
giving you the best of both worlds. Learn how to use wr as a backend for:

* :doc:`Cromwell </integrations/cromwell>`
* :doc:`Nextflow </integrations/nextflow>`

.. toctree::
   :maxdepth: 2
   :hidden:
   :caption: Integrations

   /integrations/cromwell
   /integrations/nextflow

How-to Guides
-------------

These guides will help walk you through specific use cases.

* :doc:`Execute workflows in OpenStack using S3 </guides/openstack>`

.. toctree::
   :maxdepth: 2
   :hidden:
   :caption: Guides

   /guides/openstack

.. _get-in-touch:

Get in touch
------------

The best way to report bugs or make feature requests is to `create an issue on
github <https://github.com/VertebrateResequencing/wr/issues/new>`_.

To chat with the developers and get help or discuss feature requests, join us on
`gitter <https://gitter.im/wtsi-wr>`_.

If you'd like to contribute to wr's development, follow
:doc:`this guide </advanced/contribute>`.
