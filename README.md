# Github Mirror

A simple service mirroring all public repositories of one organization from Gitlab to the organization's Github account.
The service is meant to run on the same host as the Gitlab instance.

## Installation

```sh
go get gitlab.stusta.de/stustanet/github-mirror
cp $GOPATH/src/gitlab.stusta.de/stustanet/github-mirror/etc/github-mirror.json.example /etc/github-mirror.json
```

Then adjust the `/etc/github-mirror.json`.

Tmp test