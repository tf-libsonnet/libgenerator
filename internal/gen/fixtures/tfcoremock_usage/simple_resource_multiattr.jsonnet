local tfcoremock = import './tfcoremock/main.libsonnet';
local tf = import 'github.com/tf-libsonnet/core/main.libsonnet';

local o =
  tfcoremock.simple_resource.new('foo', string='test-render-library', integer=42)
  + tf.withOutput('foo_string', o._ref.tfcoremock_simple_resource.foo.get('string'))
  + tf.withOutput('foo_integer', o._ref.tfcoremock_simple_resource.foo.get('integer'));

o
