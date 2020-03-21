# Webhooked

Simple command line utility to install a webhook in each of your GitHub repositories.

## Basic usage

1. Obtain a personal access token from https://github.com/settings/tokens

   * Webhooked requires the `admin:repo_hook` scope
   * Optional: to add hooks to private repositories, also select the `repo` scope

2. Install webhooked:

    ```
    $ go install github.com/csmith/webhooked
    ```

3. Run webhooked with your hook details:

   ```shell script
   $ webhooked \
       --token your_personal_access_token \
       --url https://example.com/your_webhook \
       --secret your_secret 
   ```

WebHooked will iterate through all the repositories you own and install the webhook.
If a repository already has a webhook with the same URL, its config will be updated
to match the one specified on the command line.

## Advanced options

### Custom event types

By default webhooked opts for your webhook to receive all events. You can specify
custom events with a comma-separated list: 

```shell script
$ webhooked \
   --token your_personal_access_token \
   --url https://example.com/your_webhook \
   --secret your_secret \
   --events push,pull_request
```

You can see all the available event types in the
[GitHub API docs for events](https://developer.github.com/v3/activity/events/types/).

### Repos owned by a different user/org

You can scan repos owned by a different org or user (that you have appropriate access to)
by specifying the owner argument:

```shell script
$ webhooked \
   --token your_personal_access_token \
   --url https://example.com/your_webhook \
   --secret your_secret \
   --owner my_org
```

If user is not specified, webhooked will scan the repositories of the user that created
the personal access token.

### Changing webhook content type

Unless otherwise specified, webhooked assumes you want JSON. To change it simply pass
the content-type argument:

```shell script
$ webhooked \
   --token your_personal_access_token \
   --url https://example.com/your_webhook \
   --secret your_secret \
   --content-type form
```

You can see the supported content types in the
[GitHub API docs for creating a hook](https://developer.github.com/v3/repos/hooks/#create-a-hook)

### Monitoring mode

Webhooked can optionally run continuously, scanning repositories at a set interval. This
is enabled with the `--monitor` flag, which takes a duration argument.

Durations are specified as a sequence of numbers with one of the following unit suffixes:
`s` for seconds, `m` for minutes, and `h` for hours. For example, `3.5h` and `3h30m` are both
three hours and thirty minutes.

Any duration under a minute will be ignored (i.e., monitor mode will be disabled). In order to
avoid an excessive amount of traffic to the GitHub API, it is recommended to set a monitor
duration of at least 1 hour, preferably much higher. If you frequently add new repositories
and require timely updates you should consider using a GitHub application instead of webhooked.

As an example:

```shell script
$ webhooked \
   --token your_personal_access_token \
   --url https://example.com/your_webhook \
   --secret your_secret \
   --monitor 6h
```

### Docker

Webhooked is available as a small docker image, and accepts all its arguments as environment
variables as well as command line arguments.

For example, to run using docker-compose:

```yaml
---
version: '3.7'

services:
  webhooked:
    image: csmith/webhooked
    environment:
      TOKEN: your_personal_access_token
      URL: https://example.com/your_webhook
      SECRET: your_secret
      MONITOR: 6h
    restart: always
```
