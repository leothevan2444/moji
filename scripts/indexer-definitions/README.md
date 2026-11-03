# Fixed Cardigann compatibility corpus

Production rules are installed at runtime. This directory contains only a lock
and a checker for repeatable development and CI validation.

Download and verify the pinned public corpus explicitly:

    python3 scripts/indexer-definitions/check_corpus.py --download

Reuse an existing archive offline:

    python3 scripts/indexer-definitions/check_corpus.py --archive /tmp/moji-indexers-v11.tar.gz --report /tmp/moji-v11-report.json

The lock records the upstream commit, archive digest and definition count.
Rules are extracted into a temporary directory and removed after checking.
The report distinguishes compile errors from optional-field diagnostics. It also
counts request features and lists affected definitions, declared character
encodings, supported query modes and referenced query variables. Feature counts
include search paths with bracket syntax in declared input names; inherited
names count only when that path enables inheritance. Feature counts
identify behavioral test candidates; they do not mark those behaviors verified.
Successful static compilation does not prove live website availability or
end-to-end task/download compatibility.

Changing the corpus pin requires inspecting schema and feature differences,
updating the archive digest/count, running the compiler and behavioral tests,
and recording unresolved compatibility differences. Do not commit a downloaded
production definition library or confidential HTTP response captures.
