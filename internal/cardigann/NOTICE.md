# Cardigann compatibility inputs

schema_v11.json is the Prowlarr Cardigann v11 format contract, retrieved from
Prowlarr/Indexers at commit 5f5db4e58d06be18619c3a858b9a91ade6f341ca:
https://github.com/Prowlarr/Indexers/blob/5f5db4e58d06be18619c3a858b9a91ade6f341ca/definitions/v11/schema.json

Production indexer definitions are downloaded at runtime, not embedded here.
Fixtures in this package are synthetic unless individually attributed.

The andmatch row-filter semantics, query flags/IMDb short ID, checkbox values
and simpleCaptcha form protocol were checked
against Jackett/Jackett at cfdc645711c487d05cb55932e394b2370154d81e:
https://github.com/Jackett/Jackett/blob/cfdc645711c487d05cb55932e394b2370154d81e/src/Jackett.Common/Indexers/Definitions/CardigannIndexer.cs
https://github.com/Jackett/Jackett/blob/cfdc645711c487d05cb55932e394b2370154d81e/src/Jackett.Common/Models/TorznabQuery.cs
Modern Prowlarr uses different host-level andmatch cleanup; the comparison
README records that boundary. These source references do not settle the
application's distribution-license review below.

Generated magnets use the public tracker defaults from that same fixed Jackett
revision, following the user's confirmed compatibility choice:
https://github.com/Jackett/Jackett/blob/cfdc645711c487d05cb55932e394b2370154d81e/src/Jackett.Common/Utils/MagnetUtil.cs
This is a host default, separate from dynamically downloaded site definitions.
The comparison verifies exact URI parameters; it does not probe those trackers'
current availability.

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

The application distribution license and upstream schema attribution must be
reviewed before release. Dependency license texts remain in their source modules.
