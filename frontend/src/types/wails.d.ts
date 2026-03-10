import * as AIService from "../../wailsjs/go/app/AIService";
import * as AgentService from "../../wailsjs/go/app/AgentService";
import * as AlertingService from "../../wailsjs/go/app/AlertingService";
import * as App from "../../wailsjs/go/app/App";
import * as BroadcastService from "../../wailsjs/go/app/BroadcastService";
import * as ComplianceService from "../../wailsjs/go/app/ComplianceService";
import * as DiscoveryService from "../../wailsjs/go/app/DiscoveryService";
import * as FileService from "../../wailsjs/go/app/FileService";
import * as HealthService from "../../wailsjs/go/app/HealthService";
import * as HostService from "../../wailsjs/go/app/HostService";
import * as LocalService from "../../wailsjs/go/app/LocalService";
import * as LogSourceService from "../../wailsjs/go/app/LogSourceService";
import * as MetricsService from "../../wailsjs/go/app/MetricsService";
import * as MultiExecService from "../../wailsjs/go/app/MultiExecService";
import * as NotesService from "../../wailsjs/go/app/NotesService";
import * as PluginService from "../../wailsjs/go/app/PluginService";
import * as RecordingService from "../../wailsjs/go/app/RecordingService";
import * as SIEMService from "../../wailsjs/go/app/SIEMService";
import * as SSHService from "../../wailsjs/go/app/SSHService";
import * as SecurityService from "../../wailsjs/go/app/SecurityService";
import * as SessionService from "../../wailsjs/go/app/SessionService";
import * as SettingsService from "../../wailsjs/go/app/SettingsService";
import * as ShareService from "../../wailsjs/go/app/ShareService";
import * as SnippetService from "../../wailsjs/go/app/SnippetService";
import * as SyncService from "../../wailsjs/go/app/SyncService";
import * as TeamService from "../../wailsjs/go/app/TeamService";
import * as TelemetryService from "../../wailsjs/go/app/TelemetryService";
import * as ThemeService from "../../wailsjs/go/app/ThemeService";
import * as TunnelService from "../../wailsjs/go/app/TunnelService";
import * as UpdaterService from "../../wailsjs/go/app/UpdaterService";
import * as VaultService from "../../wailsjs/go/app/VaultService";
import * as WorkspaceService from "../../wailsjs/go/app/WorkspaceService";


declare global {
  interface Window {
    go: {
      app: {
        AIService: typeof AIService;
        AgentService: typeof AgentService;
        AlertingService: typeof AlertingService;
        App: typeof App;
        BroadcastService: typeof BroadcastService;
        ComplianceService: typeof ComplianceService;
        DiscoveryService: typeof DiscoveryService;
        FileService: typeof FileService;
        HealthService: typeof HealthService;
        HostService: typeof HostService;
        LocalService: typeof LocalService;
        LogSourceService: typeof LogSourceService;
        MetricsService: typeof MetricsService;
        MultiExecService: typeof MultiExecService;
        NotesService: typeof NotesService;
        PluginService: typeof PluginService;
        RecordingService: typeof RecordingService;
        SIEMService: typeof SIEMService;
        SSHService: typeof SSHService;
        SecurityService: typeof SecurityService;
        SessionService: typeof SessionService;
        SettingsService: typeof SettingsService;
        ShareService: typeof ShareService;
        SnippetService: typeof SnippetService;
        SyncService: typeof SyncService;
        TeamService: typeof TeamService;
        TelemetryService: typeof TelemetryService;
        ThemeService: typeof ThemeService;
        TunnelService: typeof TunnelService;
        UpdaterService: typeof UpdaterService;
        VaultService: typeof VaultService;
        WorkspaceService: typeof WorkspaceService;

      }
    }
  }
}
export {};
