- ~how to handle local file paths for publishing~
- ~maru2 pack -> maru2 publish?~ just maru2-publish
- use defenseunicorns/pkg/oci or just oras.land?
- ~leverage an existing store, or always --fetch-all into a tempdir, then publish that~ fetch all
- how does client creation work?
  - 1 client per registry
  - ~cache the manifest so lookups don't require extra http calls?, but how much does that really gain?~ not doing this
- what entrypoints do users expect?
  - only `tasks.yaml`?
  - if chosen entrypoint, do users expect to be able to call other localpath workflows?
  - should there be an index re-export?
- are there multiarch workflows?

calculate the time complexity of every operation

prose wins over the implementation
formally define 1.0 and the stability promises
