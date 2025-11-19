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

## pipeline-null-ctx

Check for and fix issues with unnecessary null-safe operators with ctx in ingest pipelines.

```shell
package-tool analyze pipeline-null-ctx
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

## validations

Find usages of excluded validations in packages.

### Args

Note: Multiple args may be provided, separated by commas.

- `DOCS`: Filter for packages not enforcing doc standards.
- `DOCS-<title>` Filter for skipped doc standards, by title.
- `<validation>` Filter for package-spec exclude checks (i.e., SVR00004).

```shell
package-tool analyze processor-field [--args VALIDATION1,VALIDATION2]
```
