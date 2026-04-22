import type { Plugin } from "@opencode-ai/plugin";

export const WireCheckPlugin: Plugin = async ({ $, directory, client }) => {
  return {
    event: async ({ event }) => {
      if (event.type !== "session.idle") return;
      await client.app.log({
        body: {
          service: "wire-check",
          level: "info",
          message: "Session idle — starting wire validation...",
        },
      });

      // 找出所有含有 wire.go 的 di 目錄
      let wireDirs: string[] = [];
      try {
        const result =
          await $`find ${directory} -path "*/di/wire.go" -not -path "*/vendor/*"`.text();
        wireDirs = result
          .trim()
          .split("\n")
          .filter(Boolean)
          // wire.go 的路徑 → 取出所在資料夾
          .map((f) => f.replace(/\/wire\.go$/, ""));
      } catch {
        // find 沒找到任何東西時不是錯誤，直接跳過
        return;
      }

      if (wireDirs.length === 0) {
        await client.app.log({
          body: {
            service: "wire-check",
            level: "info",
            message: "No di/wire.go found, skipping.",
          },
        });
        return;
      }

      const failures: string[] = [];

      for (const dir of wireDirs) {
        // 取得相對路徑供顯示用
        const rel = dir.replace(directory + "/", "");

        await client.app.log({
          body: {
            service: "wire-check",
            level: "info",
            message: `Running wire in ${rel}`,
          },
        });

        try {
          // wire 會在當前目錄產生 wire_gen.go，需要 cd 進去執行
          await $`cd ${dir} && wire .`.quiet();

          await client.app.log({
            body: {
              service: "wire-check",
              level: "info",
              message: `✅ wire OK: ${rel}`,
            },
          });
        } catch (err: unknown) {
          const msg = err instanceof Error ? err.message : String(err);

          await client.app.log({
            body: {
              service: "wire-check",
              level: "error",
              message: `❌ wire FAILED: ${rel}`,
              extra: { error: msg },
            },
          });

          failures.push(rel);
        }
      }

      // 用 TUI toast 告知結果
      if (failures.length > 0) {
        // 顯示失敗訊息（toast 只支援純字串）
        const failList = failures.join(", ");
        await client.tui.showToast({
          body: {
            message: `⚠️  wire failed in: ${failList}`,
            variant: "error",
          },
        });
        // await client.session.prompt({
        //   path: { id: event.properties.sessionID },
        //   body: {
        //     model: { providerID: "opencode", modelID: "minimax-m2.5-free" },
        //     parts: [{ type: "text", text: "Fix the wire errors above." }],
        //   },
        // })

      } else {
        await client.tui.showToast({
          body: {
            message: `✅ All wire checks passed (${wireDirs.length} packages)`,
            variant: "success",
          },
        });
      }
    },
  };
};