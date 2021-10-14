Contribute
==========

If you'd like to help improve wr, contributions in the form of pull requests are
welcome.

If you'd like to aid in wr's development, this document provides a guide for
getting started as a new developer. It's suggested you go through this document
in order, but skip sections if you're already familiar with them.

Learn to use wr
---------------

Before developing for wr, it's useful to have the context of what it does and
how it's used.

* Go through the :doc:`basic tutorial </tutorials/basic>`.
* Try starting the manager in local mode and adding jobs.
* If you've got LSF or OpenStack environments, try starting the manager in those
  modes and adding jobs.
* If you've got an S3 bucket, try adding jobs that use wr's built-in S3
  mounting.
* If you've got an OpenStack environment, try following the
  :doc:`OpenStack tutorial </tutorials/openstack>`.

Go
--

wr is written in the Go programming language. If you're familiar with just about
any other language, including scripting languages like Perl or Python, you
should be able to get up-to-speed on Go pretty quickly.

* To learn Go, work through the `docs <https://golang.org/doc/>`_.
* In particular, try taking the `tour <https://tour.golang.org/welcome/1>`_.
* And go through the basics of `couding <https://golang.org/doc/code.html>`_.
* wr is written in idiomatic Go, so you must write
  `effectively <https://golang.org/doc/effective_go.html>`_.
* Avoid the common things brought up during
  `code reviews <https://github.com/golang/go/wiki/CodeReviewComments>`_.
* Have a read through everything linked in
  `this post <https://medium.com/@dgryski/idiomatic-go-resources-966535376dba>`_
  and aim for high quality Go code.
* You must de-lint your code before submitting it, confirmed by running
  ``make lint`` and getting no output.
* wr uses Go `modules <https://blog.golang.org/using-go-modules>`_ for
  dependency management.
* wr uses `Go Convey <https://github.com/smartystreets/goconvey/wiki>`_ for its
  tests. You should add tests for your changes, and ``make test`` and
  ``make race`` should pass afterwards.

Clean Code
----------

In addition to some of the above resources that describe how to write effective
idiomatic Go, where applicable,
`clean code <https://gist.github.com/wojteklu/73c6914cc446146b8b533c0988cf8d29>`_
ideas should be used. There's an official
`book <https://learning.oreilly.com/library/view/clean-code/9780136083238/>`_ and
video series.

Git & GitHub
------------

To get wr's source code, make changes and contribute them back, you'll need to
use Git and make GitHub pull requests.

* Start with learning the
  `basics of Git <https://git-scm.com/book/en/v2/Getting-Started-Git-Basics>`_.
* Learn about `GitHub <https://guides.github.com/activities/hello-world/>`_.
* Fork the wr repository in to your own account.
* Clone your GitHub fork to your development machine.
* Create a branch for any piece of work you start on, based on the develop
  branch.
* Add the main wr repository as an "upstream" remote, and keep your branch
  up-to-date with upstream changes by rebasing your changes on top.
* Add tests for any code you change or add.
* ``make test`` and ``make race`` should pass.
* ``make lint`` should return nothing.
* Commit and push your changes to your github fork.
* Submit a pull request to the main wr repository, being sure to compare
  against the develop branch.

What to work on
---------------

There are many open issues and a solution for any of them would be welcome. You
can use the
`project board <https://github.com/VertebrateResequencing/wr/projects/1>`_ which
lists the issues in priority order (most important at the top of each column).