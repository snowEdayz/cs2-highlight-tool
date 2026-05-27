export interface StartupState {
  mode: string;
  phase: string;
  running: boolean;
  source_step: SourceStepState;
  fatal_error: string;
  entry_notice: string;
  ads: StartupAd[];
  self_update: SelfUpdateState;
  steps: ComponentStatus[];
  can_enter_main: boolean;
  config: Record<string, unknown>;
}

export interface StartupAd {
  id: string;
  enabled: boolean;
  placement: "main_steps_top_banner";
  click_url: string;
  sponsor: string;
  title: string;
  rich_html: string;
  image_url: string;
  image_alt?: string;
}

export interface SourceStepState {
  status: string;
  source: string;
  country_code: string;
  message: string;
  error: string;
}

export interface SelfUpdateState {
  status: string;
  available: boolean;
  current: string;
  latest: string;
  url: string;
  asset_url: string;
  error: string;
}

export interface ComponentStatus {
  id: string;
  name: string;
  status: string;
  local_version: string;
  remote_version: string;
  path: string;
  error: string;
  manual_url: string;
}

export interface ProgressMessage {
  component_id: string;
  active: boolean;
  percent: number;
  indeterminate: boolean;
}
