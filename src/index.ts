import { Container, getContainer } from "@cloudflare/containers";

export class GinContainer extends Container {
  defaultPort = 8080;
  sleepAfter = "5m";
}

interface Env {
  GIN_CONTAINER: DurableObjectNamespace<GinContainer>;
}

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    const container = getContainer(env.GIN_CONTAINER);
    return container.fetch(request);
  },
};
