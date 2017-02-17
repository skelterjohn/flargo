# flargo config

## grammer

```
CONFIG -> EXECUTION*
EXECUTION -> EXECUTION_SIGNATURE EXECUTION_BODY
EXECUTION_SIGNATURE -> TYPE ':' NAME '(' [ PARAM ( ',' PARAM ) * ]
EXECUTION_BODY -> FILE_PATH
```


### working example
```
# build will begin immediately.
exec: build() build.yaml

# build will begin immediately.
exec: build_probes() probe.yaml

# "deploy_to_dev" will only start once "build" is complete. Its "out" files will
# be put in "in/build".
# This example assumpes that retagging an image is sufficient to deploy. That
# is, it assumes that the runtimes are watching for pushes to specific tags.
exec: deploy_to_dev(build) deploy_dev.yaml

# Once dev is ready, run the probes (built earlier) to check that things are
# working.
exec: test_dev(deploy_to_dev, build_probes) test_dev.yaml

# A wait directive means that a human has to run flargo to indicate this
# execution has completed. It's used as a manual gate between dev and prod,
# in this example.
wait: dev_to_prod(test_dev) -

# "deploy_to_prod" will only start once "dev_to_prod" is complete.
exec: deploy_to_prod(dev_to_prod) deploy_prod.yaml


# Once prod is ready, run the probes (built earlier) to check that things are
# working.
exec: test_prod(deploy_to_prod, build_probes) test_prod.yaml
```
