# Cardigann compatibility inputs

schema_v11.json is the Prowlarr Cardigann v11 format contract, retrieved from
Prowlarr/Indexers at commit 5f5db4e58d06be18619c3a858b9a91ade6f341ca:
https://github.com/Prowlarr/Indexers/blob/5f5db4e58d06be18619c3a858b9a91ade6f341ca/definitions/v11/schema.json

Production indexer definitions are downloaded at runtime, not embedded here.
Fixtures in this package are synthetic unless individually attributed.

The andmatch row-filter semantics, query flags/IMDb short ID, checkbox values,
single-UTF-16-character split/trim semantics, HTML entity spelling,
query-string first-value decoding, invariant casing and validate tokenization,
and simpleCaptcha form protocol were checked
against Jackett/Jackett at cfdc645711c487d05cb55932e394b2370154d81e:
https://github.com/Jackett/Jackett/blob/cfdc645711c487d05cb55932e394b2370154d81e/src/Jackett.Common/Indexers/Definitions/CardigannIndexer.cs
https://github.com/Jackett/Jackett/blob/cfdc645711c487d05cb55932e394b2370154d81e/src/Jackett.Common/Models/TorznabQuery.cs
Implicit username/password settings and the form-login method default follow
that same Cardigann constructor. An explicitly empty settings list stays empty.
Modern Prowlarr uses different host-level andmatch cleanup; the comparison
README records that boundary. These source references do not settle the
application's distribution-license review below.

Generated magnets use the public tracker defaults from that same fixed Jackett
revision, following the user's confirmed compatibility choice:
https://github.com/Jackett/Jackett/blob/cfdc645711c487d05cb55932e394b2370154d81e/src/Jackett.Common/Utils/MagnetUtil.cs
This is a host default, separate from dynamically downloaded site definitions.
The comparison verifies exact URI parameters; it does not probe those trackers'
current availability.

Future publication cleanup in Moji's BuiltinAdapter follows the user's
confirmed Jackett host policy, separate from raw Cardigann date parsing:
https://github.com/Jackett/Jackett/blob/cfdc645711c487d05cb55932e394b2370154d81e/src/Jackett.Common/Indexers/BaseIndexer.cs
It clamps future timestamps to the batch retrieval time and retains the
original rule fields used by download templates.

timeago/reltime interval behavior is checked against the same pinned Jackett
DateTimeUtil.FromTimeAgo, using its explicit relativeFrom argument for fixed
clocks. The production Go implementation uses elapsed 30-day months/365-day
years and float32 scaling; .NET remains a development-only reference.
https://github.com/Jackett/Jackett/blob/cfdc645711c487d05cb55932e394b2370154d81e/src/Jackett.Common/Utils/DateTimeUtil.cs

fuzzytime calendar branches and direct date/dateheader parsing were checked
against FromUnknown in that same source. jsonjoinarray follows the pinned
Cardigann JObject/SelectToken/child-enumeration contract. The Go ordered JSON
representation, bounded formatting and calendar handling were independently
implemented; no upstream C# runtime or source is shipped with Moji.
Clock-only fuzzy dates follow DateTimeRoutines' default-date/DateTime contract,
including AM/PM, discarded fractional seconds and retained wall-clock offsets.
The development helper fixes its default date through the public DefaultDate
property; the production Go filter uses the execution's captured date.

validfilename follows the native Path.GetInvalidFileNameChars contract, which
differs between Unix and Windows. Actual interpreter cases use the pinned Linux
reference; this filter is separate from Moji's validation of filesystem paths.

The BMP boundaries used for the IsCyrillic and IsCJKUnifiedIdeographs regular
expression aliases follow dotnet/runtime v9.0.0 RegexCharClass.cs:
https://github.com/dotnet/runtime/blob/v9.0.0/src/libraries/System.Text.RegularExpressions/src/System/Text/RegularExpressions/RegexCharClass.cs
They denote Unicode blocks rather than broader Unicode scripts.

sbcs_encoding.gen.go contains round-trip/best-fit character mappings derived
from .NET runtime v9.0.0 data under MIT. The license is reproduced verbatim in
DOTNET-LICENSE.txt. Pinned sources, checksums and regeneration instructions are
in scripts/cardigann-encodings/. The SDK is used only to reproduce test oracles;
Moji has no .NET runtime dependency. The independent runtime fixture is
testdata/dotnet-encoding-v9.json.
The Docker image carries this attribution and DOTNET-LICENSE.txt under
/usr/share/licenses/moji/cardigann/. Standalone binary packages must likewise
carry the attribution and encoding-table permission text.

The local schema compiler makes one documented compatibility amendment:
download.selectors accepts the boolean usebeforeresponse supported by the
upstream Cardigann SelectorField model. The upstream schema file is unchanged.

JSON date-to-text compatibility was independently implemented with Go's time
and regexp packages and checked against JSON.NET 13.0.3 (MIT):
https://github.com/JamesNK/Newtonsoft.Json/blob/13.0.3/Src/Newtonsoft.Json/Utilities/DateTimeParser.cs
https://github.com/JamesNK/Newtonsoft.Json/blob/13.0.3/Src/Newtonsoft.Json/Utilities/DateTimeUtils.cs
No JSON.NET source or runtime is included in Moji. The fixed actual-interpreter
tests cover the UTC-host scalar projection and its use by filters/array joins.

Implementation dependencies:

- goquery/cascadia: HTML parsing and CSS selection; BSD-3-Clause/BSD-2-Clause.
- regexp2: bounded .NET-compatible regular expressions; MIT.
- ojg: JSONPath parsing and evaluation; MIT.
- jsonschema/v6: pinned schema validation without runtime schema fetching; Apache-2.0.
- dateparse: human-readable date recognition; MIT.

At the pinned Indexers revision, no standalone LICENSE/COPYING/NOTICE file was
found in the archive. The repository's GitHub metadata also reported no license
on 2026-11-04. Its README describes automatic imports of Jackett definitions:
https://github.com/Prowlarr/Indexers/blob/5f5db4e58d06be18619c3a858b9a91ade6f341ca/README.md
Jackett's pinned repository contains GPL v2; the separately pinned Prowlarr
application contains GPL v3. These facts do not establish a separate permission
grant for every Indexers file or select an application license for Moji:
https://github.com/Jackett/Jackett/blob/cfdc645711c487d05cb55932e394b2370154d81e/LICENSE
https://github.com/Prowlarr/Prowlarr/blob/3326e2d4aa978f35fa7e44f66758ebfc8f553d67/LICENSE

The embedded schema matches the pinned archive byte-for-byte (SHA-256
376976d75daf036ead90dc55bf542ce602da8d24deb31cc10200b6bf181e3e8e).
Moji's application distribution license and authorization to redistribute that
schema remain unresolved release-review items. Runtime downloading of YAML
does not settle the embedded schema's distribution review. Dependency license
texts remain in their source modules; this input inventory does not replace
the application's complete dependency/license review.
