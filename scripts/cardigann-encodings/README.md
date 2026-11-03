# .NET single-byte encoding compatibility

The v11 corpus uses eight distinct legacy code pages (nine labels, including
the TIS-620/Windows-874 aliases). Go's strict encoders and HTML charset aliases
do not reproduce .NET's best-fit/replacement behavior. These generated tables
preserve .NET mappings without a .NET runtime dependency in Moji.

`sources.lock.json` pins the official .NET 9 data and its MIT license by SHA-256.
The generated Go file contains only round-trip/best-fit maps, about 18 KiB of
packed mapping bytes. Community indexer definitions remain runtime downloads.
Other IANA encodings continue through x/text; these oracles do not certify them.

Regenerate from already downloaded files, or explicitly allow a source download:

```sh
python3 scripts/cardigann-encodings/generate.py --source-dir /path/to/sources
python3 scripts/cardigann-encodings/generate.py --download --check
```

Sources must have the filenames in the lock. No downloaded code is executed;
the generator only parses data. Keep `DOTNET-LICENSE.txt` with the generated data.

`internal/cardigann/testdata/dotnet-encoding-v9.json` is an independent oracle
from the official .NET runtime: encoding all BMP UTF-16 values, decoding all 256
single-byte values, and samples for best fit, replacement and surrogate pairs.
Default Go tests compare the implementation with these fingerprints offline.

To reproduce the oracle, explicitly install the pinned development-only image:

```sh
docker pull mcr.microsoft.com/dotnet/sdk@sha256:6125486580cdf529a2a337d51c5557af08eaa04ef08b04f5b1d5a94f6bec7f1a
python3 scripts/cardigann-encodings/check_reference.py
```

The checker disables container networking, uses a read-only root filesystem and
mounts only disposable project files. `--write` explicitly regenerates the
fixture. It does not test Jackett's complete search or download implementation.
