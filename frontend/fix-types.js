const fs = require('fs');
const path = require('path');

const WAILS_SERVICES = [
  "AIService", "AgentService", "AlertingService", "App", "BroadcastService",
  "ComplianceService", "DiscoveryService", "FileService", "HealthService",
  "HostService", "LocalService", "LogSourceService", "MetricsService",
  "MultiExecService", "NotesService", "PluginService", "RecordingService",
  "SIEMService", "SSHService", "SecurityService", "SessionService",
  "SettingsService", "ShareService", "SnippetService", "SyncService",
  "TeamService", "TelemetryService", "ThemeService", "TunnelService",
  "UpdaterService", "VaultService", "WorkspaceService"
];

const typesDir = path.join(__dirname, 'src', 'types');
if (!fs.existsSync(typesDir)) {
  fs.mkdirSync(typesDir, { recursive: true });
}

const wailsDtsPath = path.join(typesDir, 'wails.d.ts');
let importStmts = '';
let declareStmts = '';

for (const svc of WAILS_SERVICES) {
  importStmts += `import * as ${svc} from "../../wailsjs/go/app/${svc}";\n`;
  declareStmts += `        ${svc}: typeof ${svc};\n`;
}

const dtsContent = `${importStmts}

declare global {
  interface Window {
    go: {
      app: {
${declareStmts}
      }
    }
  }
}
export {};
`;
fs.writeFileSync(wailsDtsPath, dtsContent);
console.log("Written src/types/wails.d.ts");

// Recursively get files
function walk(dir) {
  let results = [];
  const list = fs.readdirSync(dir);
  list.forEach(function(file) {
    file = path.join(dir, file);
    const stat = fs.statSync(file);
    if (stat && stat.isDirectory()) { 
      results = results.concat(walk(file));
    } else if (file.endsWith('.tsx') || file.endsWith('.ts')) { 
      results.push(file);
    }
  });
  return results;
}

const allFiles = walk(path.join(__dirname, 'src'));

for (const file of allFiles) {
  // skip the one we just wrote
  if (file === wailsDtsPath) continue;

  let content = fs.readFileSync(file, 'utf8');
  let original = content;

  // 1. Fix Wails explicit any casting
  content = content.replace(/\(window as any\)\.go\?\.app\?\./g, "window.go?.app?.");
  content = content.replace(/\(window as any\)\.go\.app\./g, "window.go.app.");
  content = content.replace(/\(window \+ as any\)/g, "window");

  // 2. Fix catch blocks
  // catch (err: any) -> catch (err: unknown)
  content = content.replace(/catch \((err|e): any\)/g, "catch ($1: unknown)");
  
  // We often do err?.message in the catch block. Since err is unknown, err?.message will TS error.
  // We can hackcast it inline just to silence the error correctly without an explicit `any` param,
  // e.g. (err as Error)?.message. 
  // Wait, let's just cast it to type `Error` inside the catch body if needed.
  // Simple regex substitution for err?.message -> (err as Error)?.message
  content = content.replace(/(\W)(err|e)\?\.message/g, "$1($2 as Error)?.message");

  // 3. Fix state initials
  content = content.replace(/createSignal<Record<string, any>>/g, "createSignal<Record<string, unknown>>");
  
  // CreateSignal<any[]> -> try to infer or fallback to Record<string, unknown>[]
  // Specifically we know WorkspacePanel uses workplaces, ThreatMap uses stats, etc.
  content = content.replace(/createSignal<any\[\]>\(\[\]\)/g, "createSignal<Record<string, unknown>[]>([])");
  content = content.replace(/createSignal<any>\((null|undefined|{})\)/g, "createSignal<unknown>($1)");

  // 4. Props & params: `(data: any)` -> `(data: unknown)` or `(data: Record<string, unknown>)`
  content = content.replace(/\(([_a-zA-Z]+):\s*any\)\s*=>/g, "($1: unknown) =>");
  content = content.replace(/:\s*any(\[\])?/g, (match, isArr) => {
      // Don't blanket replace all `any`, just a few targeted ones remaining
      return isArr ? ": Record<string,unknown>[]" : ": unknown";
  });

  if (content !== original) {
    fs.writeFileSync(file, content);
    console.log(`Updated ${file}`);
  }
}
