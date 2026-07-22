export type ShutdownResult = "graceful" | "forced";

type StoppableServer = {
  stop(closeActiveConnections?: boolean): Promise<void>;
};

export function createBoundedShutdown(server: StoppableServer, gracePeriodMilliseconds: number) {
  if (!Number.isInteger(gracePeriodMilliseconds) || gracePeriodMilliseconds < 1) {
    throw new Error("shutdown grace period must be a positive integer");
  }

  let activeShutdown: Promise<ShutdownResult> | undefined;

  return () => {
    activeShutdown ??= stopWithinGracePeriod(server, gracePeriodMilliseconds);
    return activeShutdown;
  };
}

async function stopWithinGracePeriod(
  server: StoppableServer,
  gracePeriodMilliseconds: number,
): Promise<ShutdownResult> {
  let forceTimer: ReturnType<typeof setTimeout> | undefined;
  const gracefulStop = server.stop(false).then(() => "graceful" as const);
  const forcedStop = new Promise<ShutdownResult>((resolve, reject) => {
    forceTimer = setTimeout(() => {
      server.stop(true).then(() => resolve("forced"), reject);
    }, gracePeriodMilliseconds);
  });

  try {
    return await Promise.race([gracefulStop, forcedStop]);
  } finally {
    if (forceTimer !== undefined) {
      clearTimeout(forceTimer);
    }
  }
}
