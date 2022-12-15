local tfcoremock = import './tfcoremock/main.libsonnet';
local tf = import 'github.com/fensak-io/tf-libsonnet/main.libsonnet';

local o =
  tfcoremock.simple_resource.new('foo', string='test-render-library')
  + tf.withOutput('foo_string', o._ref.tfcoremock_simple_resource.foo.get('string'));

o
