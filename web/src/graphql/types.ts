export type Task = {
  id: string;
  query: string;
  status: string;
  torrentName: string;
  progress: number;
  qbittorrentState: string;
  contentPath: string;
  stashJobId: string;
  stashScanStatus: string;
  stashScanError: string;
  createdAt: string;
  updatedAt: string;
};

export type SearchResult = {
  title: string;
  size: number;
  seeders: number;
  peers: number;
  tracker: string;
  categoryDesc: string;
  publishDate: string;
  link: string;
  magnetUri: string;
};
