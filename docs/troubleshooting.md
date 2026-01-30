# Troubleshooting

You can check the logs of the Piggy Webhooks by running the following command:

```bash
kubectl -n piggy-webhooks logs deployment/piggy-webhooks
```

The log will look like this:

```json
{"level":"info","namespace":"my-ns","owner":"my-pod","time":"2026-01-27T11:27:47Z","message":"Pod of ReplicaSet 'my-pod' has been mutated (took 30.701µs)"}
{"level":"info","namespace":"my-ns","pod_name":"my-pod","service_account":"my-ns:my-sa","secret_name":"my-ns/my-sa","time":"2026-01-27T12:30:02Z","message":"Request from [sa=my-ns:my-sa], [pod=my-pod] was successful"}
```

The first line shows that Piggy Webhooks has mutated a Pod. The second line shows that Piggy Webhooks has received a request from a Pod and the request was successful.

You can check the logs of the piggy-env by running the following command:

```bash
kubectl -n <namespace> logs <pod-name>
```

The log will look like this:

```json
{"level":"info","time":"2026-01-27T12:37:19Z","message":"Request secrets was successful"}
```

This log shows that piggy-env has requested the secrets from Piggy Webhooks and the request was successful.

## Debug mode

You can enable debug mode by setting the `PIGGY_DEBUG` environment variable in piggy-webhooks to `true`. You may set this variable by specifying `debug: true` in `values.yaml` during Helm install.

For piggy-env, you can enable debug mode by setting `piggysec.com/debug` annotation to `true` in your pod spec.

## Automatic retry and initial delay

If Piggy Webhooks fails to retrieve secrets from AWS Secrets Manager, it will retry up to `piggysec.com/piggy-number-of-retry` times with a 500ms interval. This is useful when using a service mesh like Istio where the proxy might not be ready to allow outgoing requests yet. You can also set `piggysec.com/piggy-initial-delay` to set an initial delay before starting to retrieve secrets.
