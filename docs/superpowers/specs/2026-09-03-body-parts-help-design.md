# Body Parts Help Reference Design

## Goal

Make the accepted standard `--body-parts` values discoverable in command help and in the bundled `atlas-ap-remote` skill.

## CLI help

`atlas-ap-remote submit --help` will state that `--body-parts` takes Chinese text, defaults to `全身`, and provide these reference values:

- `全身`
- `躯干部位`
- `面部（含颈部）`
- `手足`
- `头部`
- `头发`
- `口唇`
- `眼部`
- `指（趾）甲`

The CLI will continue forwarding the supplied string without local enumeration validation. This keeps the list advisory and allows newer server-supported values to work with an older CLI.

## Skill guidance

The bundled `skill/atlas-ap-remote/SKILL.md` will list the same nine reference values next to the submit workflow. It will explicitly direct the agent to pass the Chinese text rather than the numeric mapping key.

## Testing

The submit-help test will assert that all nine values and the Chinese-text guidance are present. Existing submit tests will continue to verify that the chosen text is sent unchanged as the `body_parts` multipart field.
