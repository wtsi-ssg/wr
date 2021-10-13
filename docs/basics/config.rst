Config
======

coming soon...


Running wr conf --default will print an example config file which details all
the config options available.

The default config should be fine for most people, but if you want to change
something, run wr conf --default > ~/.wr_config.yml and make changes to that.
Alternatively, as the example config file explains, add environment variables to
your shell login script and then source it. If you'll be using OpenStack, it is
strongly recommended to configure database backups to go to S3.