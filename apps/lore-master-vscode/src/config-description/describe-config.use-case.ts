import { CONFIG_DESCRIBE_METHOD, type ConfigDescription } from '../engine-protocol'

/** The one thing describing needs from the engine client: to make a request. */
export interface DescribeEngine {
  request<R> (method: string, params?: unknown): Promise<R>
}

const known = new WeakMap<DescribeEngine, Promise<ConfigDescription>>()

/**
 * Everything the engine says can be configured. Asked once per engine client; a failed ask is not
 * remembered, so the next one tries again. This is where the extension learns the platforms,
 * their fields and the settings: it keeps no list of its own.
 */
export function describeConfig (engine: DescribeEngine): Promise<ConfigDescription> {
  const remembered = known.get(engine)
  if (remembered) {
    return remembered
  }
  const asked = ask(engine)
  known.set(engine, asked)

  return asked
}

async function ask (engine: DescribeEngine): Promise<ConfigDescription> {
  try {
    return await engine.request<ConfigDescription>(CONFIG_DESCRIBE_METHOD)
  } catch (error) {
    known.delete(engine)

    throw error
  }
}
