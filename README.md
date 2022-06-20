# go-fileshare

A simple and dirty go application for sharing files on a local directory.


config.yaml
```yaml
path: dir_to_share
DataFile: data.yaml
BaseUrl: http://localhost:8080/
kutt:
  key: ....
  enabled: false
  url: https://kutt.yourdomain.com/
```

```shell
docker run --rm -ti -v $(pwd)/demo:/data -v $(pwd):/workdir -u root -p 8080:8080 ghcr.io/alexander-lindner/go-fileshare:latest
```