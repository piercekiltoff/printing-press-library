# Salesforce Headless 360 Trust Metadata

Deploy this metadata when an org edition cannot create Tooling API
`Certificate` records and must use the `SF360_Bundle_Key__mdt` fallback.

```sh
sf project deploy start --source-dir metadata --target-org <alias>
sf org assign permset --name SF360_Key_Registrar --target-org <alias>
```

After deployment, run:

```sh
salesforce-headless-360-pp-cli trust register --org <alias>
```

`trust register` prefers Certificate records. If Salesforce returns
`INVALID_TYPE` or `NOT_FOUND` for `Certificate`, it writes a CMDT key record
with a signed hash-chain receipt instead.

## Salesforce Headless 360 Write Metadata

Deploy this metadata before enabling v1.1 write commands. It adds:

- `SF360_Write_Audit__c`: a private custom object for signed write-intent audit
  rows, with an auto-number name field using `WA-{0000000000}`.
- `SF360_Idempotency_Key__c`: a unique, case-sensitive External Id text field
  on `Account`, `Contact`, `Opportunity`, `Case`, `Task`, and `Event`.
- `SF360_Key_Registrar`: field-level access for the write audit object and the
  idempotency fields, plus create/read/edit object access for
  `SF360_Write_Audit__c`.

```sh
sf project deploy start --source-dir metadata --target-org <alias>
sf org assign permset --name SF360_Key_Registrar --target-org <alias>
```

The idempotency fields are stored under the flat
`metadata/fields/<SObject>.SF360_Idempotency_Key__c.field-meta.xml` layout. This
keeps the files grouped as deployable top-level `CustomField` metadata and uses
object-qualified `fullName` values such as
`Account.SF360_Idempotency_Key__c`. Admins that prefer SFDX object folders can
move each file under `metadata/objects/<SObject>/fields/` and drop the object
prefix from `fullName`.

The audit object's requested indexed fields are represented only where metadata
has a native field-level index switch. `TargetRecordId__c` and
`IdempotencyKey__c` are External Id text fields; `ActingKid__c` and `TraceId__c`
remain plain text fields so org admins can request custom indexes separately if
their audit query patterns need them.
