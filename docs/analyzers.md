# Analyzers

## duplicate-processor

Check for processors duplicated across ingest pipelines.

```shell
package-tool analyze duplicate-processor
```

## pipeline-error

Check for and fix issues with ingest pipeline error handlers.

```shell
package-tool analyze pipeline-error
```

## pipeline-event-original

Check for and fix issues with `event.original` handling in ingest pipelines.

```shell
package-tool analyze pipeline-event-original
```

## pipeline-tag

Check for and fix issues with ingest pipeline processor tags.

```shell
package-tool analyze pipeline-tag
```

## processor-field

Check for usages of `FIELD` in ingest pipeline processors.

```shell
package-tool analyze processor-field --args FIELD
```
