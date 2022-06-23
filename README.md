# go-fileshare

A quite simple and dirty go application for sharing files on a local directory through a builtin webserver.
## Features
* Only 10.7 MB of Docker image
* Cross-platform

## Screenshots
![Image of Dolphin and the meta file content](screenshot1.png)
![Preview page of the shared file](screenshot2.png)

## How does this work?

The application is started using docker and watches (using ) a mounted and therefore a beliebige directory.
If a new file is created in the directory, a hash is generated and stored in a meta file next to the file (see the first screenshot).
The webserver provides a file and its metadata using this hash.
A simple key-value file (`data.yaml`) is used to store the hash and the file name to improve the performance.
A simple `config.yaml` file is used to configure the application.
To delete a shared file, simply delete the file (and the meta file).

An additional feature is shortify the url using a custom [Kutt](https://kutt.it) installation.

## Background

I'm running a TrueNAS Scale server with a NFS share (as well as an SMB share).
I simply want to share a file on that share with other person.
My main requirement was to only invest a couple of hours - so no fancy dolphin integration or complex web ui.
So here my solution, quite simple and very dirty ;).

## Installation

Create a `config.yaml` file somewhere with this content:
```yaml
path: dir_to_share # is ignored inside the docker container
DataFile: data.yaml
BaseUrl: http://localhost:8080/ # so the generated urls are correct
kutt: # Kutt data
  key: ....
  enabled: false
  url: https://kutt.yourdomain.com/
```
The image is based on the `distroless` image and therefore, the user `nonroot (65532:65532)` is used internally.
Set the correct file permissions for the `config.yaml` file, the `data.yaml` file and the shared dir, for example:
`chown :65532 config.yaml -R && chmod 660 config.yaml`.

Now start the container, for example by using this snippet:
```shell
docker run --rm -ti -v $(pwd)/data:/data -v $(pwd)/config:/workdir -p 8080:8080 ghcr.io/alexander-lindner/go-fileshare:latest
```
Or by using the TrueNAS Scale UI.
Add a reverse proxy in front of the container, for example for TrueNAS Scale using the `external-service` and Ingress.
