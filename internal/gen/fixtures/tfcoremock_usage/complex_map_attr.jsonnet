local tfcoremock = import './tfcoremock/main.libsonnet';
local tf = import 'github.com/fensak-io/tf-libsonnet/main.libsonnet';

tfcoremock.complex_resource.new('foo', map={
  hello: { string: 'hello' },
})
+ tfcoremock.complex_resource.withMapMixin('foo', {
  world: { string: 'world' },
})
+ tf.withOutput(
  'foo',
  '${tfcoremock_complex_resource.foo.map.hello.string} ${tfcoremock_complex_resource.foo.map.world.string}',
)
