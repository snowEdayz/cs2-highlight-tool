export type WanmeiClientStatus = "client_not_running" | "client_not_logged_in" | "ready";

export interface WanmeiMatchItem {
  match_id: string;
  download_match_id: string;
  map_name: string;
  score1: number;
  score2: number;
  kill: number;
  death: number;
  assist: number;
  k4: number;
  k5: number;
  rating: number;
  end_time: string;
}

export interface WanmeiMatchListResult {
  status: WanmeiClientStatus;
  nickname?: string;
  steam_id?: string;
  matches: WanmeiMatchItem[];
}

export interface FiveEMatchItem {
  match_id: string;
  download_match_id: string;
  map_name: string;
  score1: number;
  score2: number;
  kill: number;
  death: number;
  assist: number;
  rating: number;
  end_time: string;
}

export interface FiveEMatchListResult {
  player_name: string;
  matches: FiveEMatchItem[];
}
