# Example Input Plugin

The `example` plugin gathers metrics about example things.  This description
explains at a high level what the plugin does and provides links to where
additional information can be found.

Telegraf minimum version: Telegraf x.x Plugin minimum tested version: x.x

⭐ Telegraf v1.0.0  <!-- introduction version -->
🚩 Telegraf v1.10.0 <!-- deprecation version if any -->
🔥 Telegraf v1.20.0 <!-- removal version  if any -->
🏷️ system           <!-- plugin tags -->
💻 all              <!-- OS support -->

## Global configuration options <!-- @/docs/includes/plugin_config.md -->

In addition to the plugin-specific configuration settings, plugins support
additional global and plugin configuration settings. These settings are used to
modify metrics, tags, and field or create aliases and configure ordering, etc.
See the [CONFIGURATION.md][CONFIGURATION.md] for more details.

[CONFIGURATION.md]: ../../../docs/CONFIGURATION.md#plugins

## Configuration

```toml @sample.conf
# This is an example plugin
[[inputs.example]]
  example_option = "example_value"
```

Running `telegraf --usage <plugin-name>` also gives the sample TOML
configuration.

### example_option

A more in depth description of an option can be provided here, but only do so if
the option cannot be fully described in the sample config.

## Metrics

Here you should add an optional description and links to where the user can get
more information about the measurements.

If the output is determined dynamically based on the input source, or there are
more metrics than can reasonably be listed, describe how the input is mapped to
the output.

- measurement1
  - tags:
    - tag1 (optional description)
    - tag2
  - fields:
    - field1 (type, unit)
    - field2 (float, percent)

- measurement2
  - tags:
    - tag3
  - fields:
    - field3 (integer, bytes)
    - field4 (integer, green=1 yellow=2 red=3)
    - field5 (string)
    - field6 (float)
    - field7 (boolean)

## Sample Queries

This section can contain some useful InfluxDB queries that can be used to get
started with the plugin or to generate dashboards.  For each query listed,
describe at a high level what data is returned.

Get the max, mean, and min for the measurement in the last hour:

```sql
SELECT max(field1), mean(field1), min(field1) FROM measurement1 WHERE tag1=bar AND time > now() - 1h GROUP BY tag
```

## Troubleshooting

This optional section can provide basic troubleshooting steps that a user can
perform.

## Example Output

This section shows example output in Line Protocol format.  You can often use
`telegraf --input-filter <plugin-name> --test` or use the `file` output to get
this information.

```text
measurement1,tag1=foo,tag2=bar field1=1i,field2=2.1 1453831884664956455
measurement2,tag1=foo,tag2=bar,tag3=baz field3=1i 1453831884664956455
```
