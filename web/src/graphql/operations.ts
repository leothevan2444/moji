import { gql } from "urql";

export const DashboardQuery = gql`
  query Dashboard {
    health {
      ok
      message
    }
    version
    tasks {
      id
      query
      status
      torrentName
      progress
      qbittorrentState
      contentPath
      stashJobId
      stashScanStatus
      stashScanError
      createdAt
      updatedAt
    }
  }
`;

export const SearchQuery = gql`
  query Search($input: JackettSearchInput!) {
    jackettSearch(input: $input) {
      title
      size
      seeders
      peers
      tracker
      categoryDesc
      publishDate
      link
      magnetUri
    }
  }
`;

export const AddTorrentMutation = gql`
  mutation AddTorrent($input: QBittorrentAddInput!) {
    addTorrent(input: $input) {
      id
      status
      query
      torrentName
      progress
      stashScanStatus
      createdAt
    }
  }
`;

export const SyncTaskProgressMutation = gql`
  mutation SyncTaskProgress {
    syncTaskProgress {
      id
      status
      progress
      qbittorrentState
      updatedAt
    }
  }
`;

export const TriggerStashScansMutation = gql`
  mutation TriggerStashScans {
    triggerStashScans {
      id
      stashJobId
      stashScanStatus
      stashScanError
      updatedAt
    }
  }
`;
