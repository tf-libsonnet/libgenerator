local tfcoremock = import './tfcoremock/main.libsonnet';
local tf = import 'github.com/tf-libsonnet/core/main.libsonnet';

local o =
  tfcoremock.simple_resource.new('foo', string='test-render-library', _meta=tf.meta.new(count=5))
  + tf.withOutput('num_resources', '${length(tfcoremock_simple_resource.foo)}');

o
