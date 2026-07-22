import {describe, expect, test} from "bun:test";
import {createBoundedShutdown} from "./server-lifecycle";

describe("bounded Bun server shutdown", () => {
  test("stops gracefully while requests drain within the deadline", async () => {
    const calls: boolean[] = [];
    const shutdown = createBoundedShutdown({
      stop(closeActiveConnections = false) {
        calls.push(closeActiveConnections);
        return Promise.resolve();
      },
    }, 25);

    expect(await shutdown()).toBe("graceful");
    expect(calls).toEqual([false]);
  });

  test("forces active connections closed after the bounded grace period", async () => {
    const calls: boolean[] = [];
    const shutdown = createBoundedShutdown({
      stop(closeActiveConnections = false) {
        calls.push(closeActiveConnections);
        return closeActiveConnections ? Promise.resolve() : new Promise<void>(() => undefined);
      },
    }, 5);

    expect(await shutdown()).toBe("forced");
    expect(calls).toEqual([false, true]);
  });

  test("coalesces repeated termination signals into one shutdown", async () => {
    const calls: boolean[] = [];
    let finishGracefully: (() => void) | undefined;
    const gracefulStop = new Promise<void>((resolve) => {
      finishGracefully = resolve;
    });
    const shutdown = createBoundedShutdown({
      stop(closeActiveConnections = false) {
        calls.push(closeActiveConnections);
        return gracefulStop;
      },
    }, 25);

    const first = shutdown();
    const second = shutdown();
    expect(second).toBe(first);
    finishGracefully?.();
    expect(await first).toBe("graceful");
    expect(calls).toEqual([false]);
  });
});
