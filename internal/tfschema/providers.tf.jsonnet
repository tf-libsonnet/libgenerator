local mergeAll(objs) = std.foldl(
  function(x, y) (x + y),
  objs,
  {},
);


// Accept the list of providers as a TLA json object and render out the required_providers block so that all the
// providers can be downloaded.
function(providers)
  mergeAll([
    {
      terraform+: {
        required_providers+: {
          [p.name]: {
            source: p.src,
            version: p.version,
          },
        },
      },
    }
    for p in providers
  ])
