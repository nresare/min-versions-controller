
## Debugging TLS

To get a shell with curl in the current cluster, use
`kubectl run -it dbg --image=nresare/dbg --rm -- sh`

Once inside, you can verify the TLS setup with
```shell
$ cat > ca.crt
<paste from certs/ca.crt>
ctrl-d
$ curl -v --cacert ./ca.crt https://podhook.min-versions.svc
```

## Tailing the controller logs

```shell
$ logs -f -l app.kubernetes.io/name=min-versions-controller -n min-versions
```