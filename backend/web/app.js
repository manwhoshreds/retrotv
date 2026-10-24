let movies = [];
let selectedMovie = null;
let tvShows = [];
let selectedTVShow = null;
let moviesLoaded = false;
let tvShowsLoaded = false;
let currentView = "movies";
let moviePage = 1;
let tvShowPage = 1;
let tvShowDetailOpen = false;
let selectedTVSeason = null;
const libraryPageSize = 48;
let movieLibraryMode = "all";
let movieSortMode = "title-asc";
let guideChannels = [];
let guidePrograms = [];
let selectedGuideProgram = null;
let guideLoaded = false;
let guideDateValue = localDateKey(new Date());
let guideLoadSequence = 0;
const guideDayCache = new Map();
const guidePixelsPerHour = 200;
let collections = [];
let editingCollection = null;
let selectedCollection = null;
let collectionPinnedMedia = [];
let collectionExcludedMedia = [];
let collectionPreviewItems = [];
let collectionPreviewTotal = 0;
let collectionArtworkPath = "";
let collectionArtworkMode = "auto";
let collectionArtworkMedia = [];
let collectionPreviewTimer = null;
const collectionPreviewCache = new Map();
const collectionPreviewRequests = new Map();

let channels = [];
let currentChannel = null;
let mediaSources = [];
let assetSources = [];
let channelAssets = [];
let streamingMode = "direct";
let draggedChannelID = null;
let channelImportStatusTimer = null;
let builderLogoPreviewURL = null;
let builderLogoPreviewVersion = 0;
let activeBuilderTab = "content";
let programmingBlocks = [];
let programmingEvents = [];
let selectedProgrammingType = "block";
let selectedProgrammingID = null;
let pendingProgrammingBlockDays = ["mon", "tue", "wed", "thu", "fri"];
let pendingProgrammingBlockMedia = [];
let dashboardNowPlaying = [];
let dashboardStorage = null;
let programmingPageChannelID = null;
let programmingPageBlocks = [];
let programmingPageSelectedBlockID = null;
let programmingWeekOffset = 0;
let programmingSourceTab = "collections";
let programmingWeekPickerMonth = null;
let pendingProgrammingDrop = null;
let pendingProgrammingDropDays = [];
let pendingProgrammingRecurrenceAdjustment = null;
let programmingDropScheduleCheckTimer = null;
let programmingDropScheduleCheckSequence = 0;

const grid = document.querySelector("#movie-grid");
const detailsPanel = document.querySelector("#details-panel");
const search = document.querySelector("#search");
const movieCount = document.querySelector("#movie-count");
const moviePagination = document.querySelector("#movie-pagination");
const tvGrid = document.querySelector("#tv-grid");
const tvDetailsPanel = document.querySelector("#tv-details-panel");
const tvShowFullDetail = document.querySelector("#tv-show-full-detail");
const tvLibraryPane = document.querySelector("#tv-view .library-pane");
const tvSearch = document.querySelector("#tv-search");
const tvCount = document.querySelector("#tv-count");
const tvPagination = document.querySelector("#tv-pagination");
const scanTVButton = document.querySelector("#scan-tv-button");
const tvScanStatus = document.querySelector("#tv-scan-status");
const scanButton = document.querySelector("#scan-button");
const refreshSelectedMetadataButton = document.querySelector("#refresh-selected-metadata");
const movieSort = document.querySelector("#movie-sort");
const movieFilterToggle = document.querySelector("#movie-filter-toggle");
const movieGenreFilter = document.querySelector("#movie-genre-filter");
const movieYearFilter = document.querySelector("#movie-year-filter");
const movieRatingFilter = document.querySelector("#movie-rating-filter");
const movieRuntimeFilter = document.querySelector("#movie-runtime-filter");
const clearMovieFiltersButton = document.querySelector("#clear-movie-filters");
const tvGenreFilter = document.querySelector("#tv-genre-filter");
const tvYearFilter = document.querySelector("#tv-year-filter");
const tvRatingFilter = document.querySelector("#tv-rating-filter");
const tvNetworkFilter = document.querySelector("#tv-network-filter");
const tvFilterToggle = document.querySelector("#tv-filter-toggle");
const libraryFilterLinks = document.querySelectorAll("[data-library-mode]");
const guideDate = document.querySelector("#guide-date");
const guideTodayButton = document.querySelector("#guide-today");
const guidePreviousDayButton = document.querySelector("#guide-previous-day");
const guideNextDayButton = document.querySelector("#guide-next-day");
const guideNowButton = document.querySelector("#guide-now");
const guideStatus = document.querySelector("#guide-status");
const guideTimeHeader = document.querySelector("#guide-time-header");
const guideSchedule = document.querySelector("#guide-schedule");
const guideInspector = document.querySelector("#guide-inspector");
const collectionsHeader = document.querySelector(".collections-header");
const collectionsWorkspace = document.querySelector(".collections-workspace");
const collectionGrid = document.querySelector("#collection-grid");
const collectionsEmptyState = document.querySelector("#collections-empty-state");
const collectionCount = document.querySelector("#collection-count");
const createCollectionButton = document.querySelector("#create-collection-button");
const createFirstCollectionButton = document.querySelector("#create-first-collection-button");
const collectionEditor = document.querySelector("#collection-editor");
const collectionEditorTitle = document.querySelector("#collection-editor-title");
const collectionBackButton = document.querySelector("#collection-back-button");
const cancelCollectionButton = document.querySelector("#cancel-collection-button");
const saveCollectionButton = document.querySelector("#save-collection-button");
const collectionNameInput = document.querySelector("#collection-name");
const collectionDescriptionInput = document.querySelector("#collection-description");
const collectionRules = document.querySelector("#collection-rules");
const addCollectionRuleButton = document.querySelector("#add-collection-rule");
const collectionMatchSummary = document.querySelector("#collection-match-summary");
const collectionMediaSearch = document.querySelector("#collection-media-search");
const collectionMediaResults = document.querySelector("#collection-media-results");
const collectionPinnedMediaContainer = document.querySelector("#collection-pinned-media");
const collectionArtworkPreview = document.querySelector("#collection-artwork-preview");
const collectionArtworkFile = document.querySelector("#collection-artwork-file");
const chooseCollectionArtwork = document.querySelector("#choose-collection-artwork");
const removeCollectionArtwork = document.querySelector("#remove-collection-artwork");
const shuffleCollectionArtwork = document.querySelector("#shuffle-collection-artwork");
const collectionSaveStatus = document.querySelector("#collection-save-status");
const collectionInspector = document.querySelector("#collection-inspector");
const collectionPreviewSearch = document.querySelector("#collection-preview-search");
const collectionPreviewShowExcluded = document.querySelector("#collection-preview-show-excluded");
const collectionPreviewList = document.querySelector("#collection-preview-list");
const collectionPreviewCount = document.querySelector("#collection-preview-count");
const saveSettingsButton = document.querySelector("#save-settings-button");
const saveSettingsLabel = document.querySelector("#save-settings-label");
const saveSettingsSpinner = document.querySelector(".settings-save-spinner");
const scanStatus = document.querySelector("#scan-status");
const sourceList = document.querySelector("#source-list");
const addSourceButton = document.querySelector("#add-source-button");
const assetSourceList = document.querySelector("#asset-source-list");
const addAssetSourceButton = document.querySelector("#add-asset-source-button");
const scanAssetsButton = document.querySelector("#scan-assets-button");
const assetSummary = document.querySelector("#asset-summary");
const settingsStatus = document.querySelector("#settings-status");
let settingsDirty = false;
const streamingModeControl = document.querySelector("#streaming-mode-control");
const scanAllSourcesButton = document.querySelector("#scan-all-sources-button");
const forceMetadataScan = document.querySelector("#force-metadata-scan");
const scanProgressBar = document.querySelector("#scan-progress-bar");
const scanProgressPhase = document.querySelector("#scan-progress-phase");
const scanProgressCurrent = document.querySelector("#scan-progress-current");
const scanProgressCount = document.querySelector("#scan-progress-count");
const scanProgressErrors = document.querySelector("#scan-progress-errors");
const scanErrorModal = document.querySelector("#scan-error-modal");
const scanErrorTitle = document.querySelector("#scan-error-title");
const scanErrorSummaryText = document.querySelector("#scan-error-summary");
const scanErrorList = document.querySelector("#scan-error-list");
const scanErrorClose = document.querySelector("#scan-error-close");
const channelSearch = document.querySelector("#channel-search");
const channelRows = document.querySelector("#channel-rows");
const channelCount = document.querySelector("#channel-count");
const channelImportProgress = document.querySelector("#channel-import-progress");
const channelImportProgressText = document.querySelector("#channel-import-progress-text");
const createChannelButton = document.querySelector("#create-channel-button");
const exportChannelsButton = document.querySelector("#export-channels-button");
const importChannelsButton = document.querySelector("#import-channels-button");
const importChannelsFileInput = document.querySelector("#import-channels-file");
const saveChannelButton = document.querySelector("#save-channel-button");
const channelSaveStatus = document.querySelector("#channel-save-status");
const channelNameInput = document.querySelector("#channel-name");
const channelDescriptionInput = document.querySelector("#channel-description");
const channelLogoPathInput = document.querySelector("#channel-logo-path");
const channelLogoFileInput = document.querySelector("#channel-logo-file");
const channelLogoEditButton = document.querySelector("#channel-logo-edit-button");
const channelContentTypeInput = document.querySelector("#channel-content-type");
const builderLogoPreview = document.querySelector("#builder-logo-preview");
const addRuleButton =
  document.querySelector("#add-rule-button") || document.querySelector(".content-card-head .button.primary");
const rulesList = document.querySelector("#rules-list") || document.querySelector(".rules-list");
const ruleCountSummary = document.querySelector("#rule-count-summary");
const builderTabButtons = document.querySelectorAll("[data-builder-tab]");
const builderTabPanels = document.querySelectorAll("[data-builder-panel]");
const addProgrammingBlockButton = document.querySelector("#add-programming-block-button");
const addProgrammingEventButton = document.querySelector("#add-programming-event-button");
const programmingBlockList = document.querySelector("#programming-block-list");
const programmingEventList = document.querySelector("#programming-event-list");
const programmingEditor = document.querySelector("#programming-editor");
const programmingBlockModal = document.querySelector("#programming-block-modal");
const programmingBlockTitleInput = document.querySelector("#programming-block-title");
const programmingBlockWeekdaysButton = document.querySelector("#programming-block-weekdays");
const programmingBlockDays = document.querySelector("#programming-block-days");
const programmingBlockStartInput = document.querySelector("#programming-block-start");
const programmingBlockStopInput = document.querySelector("#programming-block-stop");
const programmingBlockMediaList = document.querySelector("#programming-block-media-list");
const programmingBlockMediaCount = document.querySelector("#programming-block-media-count");
const cancelProgrammingBlock = document.querySelector("#cancel-programming-block");
const cancelProgrammingBlockX = document.querySelector("#cancel-programming-block-x");
const createProgrammingBlockButton = document.querySelector("#create-programming-block");
const deleteRuleModal = document.querySelector("#delete-rule-modal");
const cancelDeleteRule = document.querySelector("#cancel-delete-rule");
const cancelDeleteRuleX = document.querySelector("#cancel-delete-rule-x");
const confirmDeleteRule = document.querySelector("#confirm-delete-rule");
const deleteChannelModal = document.querySelector("#delete-channel-modal");
const deleteChannelMessage = document.querySelector("#delete-channel-message");
const cancelDeleteChannel = document.querySelector("#cancel-delete-channel");
const cancelDeleteChannelX = document.querySelector("#cancel-delete-channel-x");
const confirmDeleteChannel = document.querySelector("#confirm-delete-channel");
const navItems = document.querySelectorAll("[data-view]");
const pages = document.querySelectorAll(".page");
const dashboardClock = document.querySelector("#dashboard-clock");
const dashboardChannelCount = document.querySelector("#dashboard-channel-count");
const dashboardMovieCount = document.querySelector("#dashboard-movie-count");
const dashboardShowCount = document.querySelector("#dashboard-show-count");
const dashboardEpisodeCount = document.querySelector("#dashboard-episode-count");
const dashboardStorageUsed = document.querySelector("#dashboard-storage-used");
const dashboardStorageTotal = document.querySelector("#dashboard-storage-total");
const dashboardRandomMovies = document.querySelector("#dashboard-random-movies");
const dashboardSpecialList = document.querySelector("#dashboard-special-list");
const programmingChannelSelect = document.querySelector("#programming-channel-select");
const programmingManageChannel = document.querySelector("#programming-manage-channel");
const programmingWeekLabel = document.querySelector("#programming-week-label");
const programmingPreviousWeek = document.querySelector("#programming-previous-week");
const programmingNextWeek = document.querySelector("#programming-next-week");
const programmingWeekPicker = document.querySelector("#programming-week-picker");
const programmingPickerMonth = document.querySelector("#programming-picker-month");
const programmingPickerWeeks = document.querySelector("#programming-picker-weeks");
const programmingPreviousMonth = document.querySelector("#programming-previous-month");
const programmingNextMonth = document.querySelector("#programming-next-month");
const programmingPickerToday = document.querySelector("#programming-picker-today");
const programmingPublish = document.querySelector("#programming-publish");
const programmingCollectionSearch = document.querySelector("#programming-collection-search");
const programmingCollectionType = document.querySelector("#programming-collection-type");
const programmingCollectionList = document.querySelector("#programming-collection-list");
const programmingCalendar = document.querySelector("#programming-calendar");
const programmingAllDayLane = document.querySelector("#programming-all-day-lane");
const programmingInspector = document.querySelector("#programming-inspector");
const programmingNewCollection = document.querySelector("#programming-new-collection");
const programmingSourceTabs = document.querySelectorAll("[data-programming-source]");
const programmingDropModal = document.querySelector("#programming-drop-modal");
const programmingDropSource = document.querySelector("#programming-drop-source");
const programmingDropWeekly = document.querySelector("#programming-drop-weekly");
const programmingDropDays = document.querySelector("#programming-drop-days");
const programmingDropDateField = document.querySelector("#programming-drop-date-field");
const programmingDropDate = document.querySelector("#programming-drop-date");
const programmingDropStart = document.querySelector("#programming-drop-start");
const programmingDropEnd = document.querySelector("#programming-drop-end");
const programmingDropPlayback = document.querySelector("#programming-drop-playback");
const programmingDropPlaybackField = document.querySelector("#programming-drop-playback-field");
const programmingDropError = document.querySelector("#programming-drop-error");
const programmingScheduleNotice = document.querySelector("#programming-schedule-notice");
const programmingDropClose = document.querySelector("#programming-drop-close");
const programmingDropCancel = document.querySelector("#programming-drop-cancel");
const programmingDropConfirm = document.querySelector("#programming-drop-confirm");
const programmingRecurrenceModal = document.querySelector("#programming-recurrence-modal");
const programmingRecurrenceClose = document.querySelector("#programming-recurrence-close");
const programmingRecurrenceCancel = document.querySelector("#programming-recurrence-cancel");
let pendingDeleteRule = null;
let pendingDeleteChannel = null;
let scanProgressTimer = null;
let scanProgressHideTimer = null;
let scanProgressController = null;
const ruleOptions = {
  Title: {
    field: "title",
    operators: [
      { value: "is", label: "is" },
    ],
    values: [],
    icon: "title",
  },
  Genre: {
    field: "genre",
    operators: [
      { value: "is", label: "is" },
      { value: "is_not", label: "is not" },
    ],
    values: [],
    icon: "genre",
  },
  Year: {
    field: "year",
    operators: [
      { value: "equals", label: "equals" },
      { value: "greater_than", label: "greater than" },
      { value: "less_than", label: "less than" },
    ],
    values: [],
    icon: "calendar",
  },
  Runtime: {
    field: "runtime",
    operators: [
      { value: "equals", label: "equals" },
      { value: "greater_than", label: "greater than" },
      { value: "less_than", label: "less than" },
    ],
    values: [],
    icon: "runtime",
  },
  Rating: {
    field: "rating",
    operators: [
      { value: "is", label: "is" },
      { value: "is_not", label: "is not" },
    ],
    values: [],
    icon: "rating",
  },
  Studio: {
    field: "studio",
    operators: [
      { value: "is", label: "is" },
      { value: "is_not", label: "is not" },
    ],
    values: [],
    icon: "studio",
  },
  Actor: {
    field: "actor",
    operators: [
      { value: "is", label: "is" },
      { value: "is_not", label: "is not" },
    ],
    values: [],
    icon: "person",
  },
  Director: {
    field: "director",
    operators: [
      { value: "is", label: "is" },
      { value: "is_not", label: "is not" },
    ],
    values: [],
    icon: "person",
  },
  Writer: {
    field: "writer",
    operators: [
      { value: "is", label: "is" },
      { value: "is_not", label: "is not" },
    ],
    values: [],
    icon: "title",
  },
  Country: {
    field: "country",
    operators: [
      { value: "is", label: "is" },
      { value: "is_not", label: "is not" },
    ],
    values: [],
    icon: "network",
  },
  Language: {
    field: "language",
    operators: [
      { value: "is", label: "is" },
      { value: "is_not", label: "is not" },
    ],
    values: [],
    icon: "title",
  },
  Quality: {
    field: "quality",
    operators: [
      { value: "is", label: "is" },
      { value: "is_not", label: "is not" },
    ],
    values: [
      { value: "hd", label: "HD (720p+)" },
      { value: "sd", label: "SD" },
    ],
    fixedValues: true,
    icon: "rating",
  },
  "Series Status": {
    field: "series_status",
    operators: [
      { value: "is", label: "is" },
      { value: "is_not", label: "is not" },
    ],
    values: [],
    icon: "rating",
  },
  Tags: {
    field: "tag",
    operators: [
      { value: "is", label: "is" },
      { value: "is_not", label: "is not" },
    ],
    values: [],
    icon: "genre",
  },
  Network: {
    field: "network",
    operators: [
      { value: "is", label: "is" },
      { value: "is_not", label: "is not" },
    ],
    values: [],
    icon: "network",
  },
  "First Air Date": {
    field: "first_air_date",
    operators: [
      { value: "equals", label: "equals" },
      { value: "greater_than", label: "after" },
      { value: "less_than", label: "before" },
    ],
    values: [],
    icon: "calendar",
  },
  "Date Added": {
    field: "date_added",
    operators: [
      { value: "equals", label: "equals" },
      { value: "greater_than", label: "after" },
      { value: "less_than", label: "before" },
    ],
    values: [],
    icon: "calendar",
  },
  "Release Date": {
    field: "release_date",
    operators: [
      { value: "equals", label: "equals" },
      { value: "greater_than", label: "after" },
      { value: "less_than", label: "before" },
    ],
    values: [],
    icon: "calendar",
  },
};
const ruleMediaTypes = [
  { value: "movie", label: "Movies", icon: "movie" },
  { value: "tv", label: "TV Shows", icon: "tv" },
];
const ruleFieldsByMediaType = {
  movie: [
    "Title",
    "Genre",
    "Year",
    "Runtime",
    "Rating",
    "Studio",
    "Actor",
    "Director",
    "Writer",
    "Country",
    "Language",
    "Release Date",
    "Date Added",
    "Tags",
    "Quality",
  ],
  tv: [
    "Title",
    "Genre",
    "Year",
    "Runtime",
    "Rating",
    "Network",
    "Studio",
    "Actor",
    "Director",
    "Writer",
    "Country",
    "Language",
    "Series Status",
    "First Air Date",
    "Date Added",
    "Tags",
    "Quality",
  ],
};
const ruleIcons = {
  movie:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><rect x="4" y="4" width="16" height="16" rx="2" /><path d="M9 4v16M15 4v16M4 9h5M15 9h5M4 15h5M15 15h5" /></svg>',
  tv:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><rect x="3" y="5" width="18" height="13" rx="2" /><path d="M8 21h8M12 18v3" /></svg>',
  title:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 7h16" /><path d="M9 7v12" /><path d="M15 7v12" /><path d="M6 19h6M12 19h6" /></svg>',
  genre:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="m4 12 8-4 8 4-8 4-8-4Z" /><path d="m4 16 8 4 8-4" /><path d="m4 8 8-4 8 4" /></svg>',
  calendar:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><rect x="4" y="5" width="16" height="16" rx="2" /><path d="M16 3v4M8 3v4M4 11h16" /></svg>',
  rating:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="m12 3 2.8 5.7 6.2.9-4.5 4.4 1.1 6.2-5.6-3-5.6 3 1.1-6.2L3 9.6l6.2-.9Z" /></svg>',
  studio:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 20V8l4 3V8l4 3V8l4 3V4h4v16Z" /><path d="M7 17h2M11 17h2M15 17h2" /></svg>',
  network:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="2" /><path d="M12 4v4M12 16v4M4 12h4M16 12h4" /><path d="m6.3 6.3 2.8 2.8M14.9 14.9l2.8 2.8M17.7 6.3l-2.8 2.8M9.1 14.9l-2.8 2.8" /></svg>',
  person:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="8" r="4" /><path d="M4 21c1.4-4 4-6 8-6s6.6 2 8 6" /></svg>',
  runtime:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="9" /><path d="M12 7v5l3 2" /></svg>',
  certification:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 3 5 6v6c0 4.5 3 7.5 7 9 4-1.5 7-4.5 7-9V6Z" /><path d="m9 12 2 2 4-4" /></svg>',
};

function setView(view) {
  if (!document.querySelector(`#${view}-view`)) {
    return;
  }

  currentView = view;
  document.body.dataset.view = view;
  pages.forEach((page) => page.classList.toggle("active", page.id === `${view}-view`));
  document.querySelectorAll(".nav-item").forEach((item) => {
    item.classList.toggle("active", item.dataset.view === view || (view === "channel-builder" && item.dataset.view === "channels"));
  });
  document.querySelectorAll(".app-nav-item").forEach((item) => {
    const section = item.dataset.view;
    item.classList.toggle("active", section === view || (section === "movies" && ["tv", "collections"].includes(view)));
  });
  if (view === "guide" && !guideLoaded) {
    loadGuide();
  }
  if (view === "dashboard") {
    renderDashboard();
  }
  if (view === "programming") {
    renderProgrammingPage();
  }
}

function formatDashboardBytes(bytes) {
  const value = Number(bytes || 0);
  if (!value) return "—";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const unit = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1);
  return `${(value / (1024 ** unit)).toFixed(unit >= 3 ? 2 : 0)} ${units[unit]}`;
}

function dashboardSpecials() {
  return channels.flatMap((channel) => (channel.programming_events || []).map((event) => ({ ...event, channel })))
    .filter((event) => event.active !== false)
    .sort((left, right) => String(left.time || "").localeCompare(String(right.time || "")))
    .slice(0, 5);
}

function renderDashboard() {
  if (!dashboardChannelCount) return;
  dashboardChannelCount.textContent = channels.length.toLocaleString();
  dashboardMovieCount.textContent = movies.length.toLocaleString();
  dashboardShowCount.textContent = tvShows.length.toLocaleString();
  dashboardEpisodeCount.textContent = tvShows.reduce((total, show) => total + Number(show.localEpisodeCount || show.episodes || 0), 0).toLocaleString();
  if (dashboardStorage) {
    dashboardStorageUsed.textContent = formatDashboardBytes(dashboardStorage.used_bytes);
    dashboardStorageTotal.textContent = `of ${formatDashboardBytes(dashboardStorage.total_bytes)}`;
  }

  const featured = dashboardNowPlaying;
  dashboardRandomMovies.innerHTML = featured.length > 0 ? featured.map((item) => {
    const movie = item.movie;
    const progress = Math.max(0, Math.min(100, Math.round(((Date.now() - item.start.getTime()) / (item.end.getTime() - item.start.getTime())) * 100)));
    return `
    <article class="dashboard-featured-movie" data-dashboard-movie="${escapeAttribute(movie.id)}" tabindex="0" role="button">
      <div class="dashboard-featured-poster" style="--poster:${escapeAttribute(movie.poster)}"></div>
      <div class="dashboard-featured-copy">
        <small class="dashboard-featured-kicker">CH ${String(item.channelNumber).padStart(2, "0")} • ${escapeHTML(item.channel.name)}</small>
        <h3>${escapeHTML(movie.title)}</h3>
        <p>${escapeHTML([movie.year, movie.rating, movie.runtime].filter(Boolean).join(" • "))}</p>
        <div class="dashboard-progress-meta"><span>${item.start.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}</span><span>${item.end.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}</span></div>
        <div class="dashboard-progress" style="--progress:${progress}%"><span></span></div>
      </div>
      <span class="dashboard-live">LIVE</span>
    </article>`;
  }).join("") : `<p class="dashboard-empty">No movies are currently airing on any channel.</p>`;
  dashboardRandomMovies.querySelectorAll("[data-dashboard-movie]").forEach((card) => {
    const openMovie = () => {
      const movie = movies.find((item) => String(item.id) === card.dataset.dashboardMovie);
      if (!movie) return;
      selectedMovie = movie;
      setView("movies");
      render();
    };
    card.addEventListener("click", openMovie);
    card.addEventListener("keydown", (event) => {
      if (event.key === "Enter" || event.key === " ") openMovie();
    });
  });

  const specials = dashboardSpecials();
  dashboardSpecialList.innerHTML = specials.length > 0 ? specials.map((event) => `
    <article class="dashboard-special"><time>${escapeHTML(event.time || "TBA")}</time><div><strong>${escapeHTML(event.name || "Special Event")}</strong><small>${escapeHTML(event.program || "Programming special")}</small></div><span>${escapeHTML(event.channel.name || "CHANNEL")}</span></article>`).join("")
    : `<p class="dashboard-empty">No upcoming special events are scheduled.</p>`;
}

async function loadDashboardNowPlaying() {
  const now = new Date();
  const eligibleChannels = channels
    .map((channel, index) => ({ channel, channelNumber: index + 1 }))
    .filter(({ channel }) => channel.airing !== false);
  const schedules = await Promise.all(eligibleChannels.map(async ({ channel, channelNumber }) => {
    try {
      const response = await fetch(`/api/channels/${channel.id}/schedule?current=1`);
      if (!response.ok) return null;
      const payload = await response.json();
      const program = (payload.programs || []).find((entry) => {
        const start = new Date(entry.start_time);
        const end = new Date(entry.end_time);
        return entry.media?.type === "movie" && start <= now && end > now;
      });
      if (!program) return null;
      const movie = movies.find((entry) => Number(entry.id) === Number(program.media.id));
      if (!movie) return null;
      return { channel, channelNumber, movie, start: new Date(program.start_time), end: new Date(program.end_time) };
    } catch (_) {
      return null;
    }
  }));
  dashboardNowPlaying = schedules.filter(Boolean).slice(0, 2);
  renderDashboard();
}

async function loadDashboardStorage() {
  try {
    const response = await fetch("/api/dashboard/storage");
    if (!response.ok) throw new Error("Storage unavailable");
    dashboardStorage = await response.json();
    renderDashboard();
  } catch (_) {
    dashboardStorage = null;
    if (dashboardStorageTotal) dashboardStorageTotal.textContent = "Storage unavailable";
  }
}

function programmingWeekStart() {
  const date = new Date();
  const day = (date.getDay() + 6) % 7;
  date.setHours(0, 0, 0, 0);
  date.setDate(date.getDate() - day + programmingWeekOffset * 7);
  return date;
}

function mondayForDate(value) {
  const date = new Date(value);
  date.setHours(0, 0, 0, 0);
  date.setDate(date.getDate() - ((date.getDay() + 6) % 7));
  return date;
}

function setProgrammingWeek(date) {
  const selected = mondayForDate(date);
  const current = mondayForDate(new Date());
  const selectedUTC = Date.UTC(selected.getFullYear(), selected.getMonth(), selected.getDate());
  const currentUTC = Date.UTC(current.getFullYear(), current.getMonth(), current.getDate());
  programmingWeekOffset = Math.round((selectedUTC - currentUTC) / (7 * 24 * 60 * 60 * 1000));
  closeProgrammingWeekPicker();
  renderProgrammingPage();
}

function renderProgrammingWeekPicker() {
  if (!programmingWeekPickerMonth) {
    const selected = programmingWeekStart();
    programmingWeekPickerMonth = new Date(selected.getFullYear(), selected.getMonth(), 1);
  }
  programmingPickerMonth.textContent = programmingWeekPickerMonth.toLocaleDateString([], { month: "long", year: "numeric" });
  const first = new Date(programmingWeekPickerMonth.getFullYear(), programmingWeekPickerMonth.getMonth(), 1);
  const gridStart = mondayForDate(first);
  const selectedKey = localDateKey(programmingWeekStart());
  const todayKey = localDateKey(new Date());
  programmingPickerWeeks.innerHTML = Array.from({ length: 6 }, (_, weekIndex) => {
    const weekStart = new Date(gridStart); weekStart.setDate(weekStart.getDate() + weekIndex * 7);
    const days = Array.from({ length: 7 }, (_, dayIndex) => {
      const date = new Date(weekStart); date.setDate(date.getDate() + dayIndex);
      const classes = [date.getMonth() !== programmingWeekPickerMonth.getMonth() ? "outside" : "", localDateKey(date) === todayKey ? "today" : ""].filter(Boolean).join(" ");
      return `<span${classes ? ` class="${classes}"` : ""}>${date.getDate()}</span>`;
    }).join("");
    return `<button class="programming-picker-week${localDateKey(weekStart) === selectedKey ? " selected" : ""}" type="button" data-picker-week="${localDateKey(weekStart)}" aria-label="Week of ${escapeAttribute(weekStart.toLocaleDateString())}">${days}</button>`;
  }).join("");
  programmingPickerWeeks.querySelectorAll("[data-picker-week]").forEach((row) => row.addEventListener("click", () => setProgrammingWeek(new Date(`${row.dataset.pickerWeek}T00:00:00`))));
}

function openProgrammingWeekPicker() {
  const selected = programmingWeekStart();
  programmingWeekPickerMonth = new Date(selected.getFullYear(), selected.getMonth(), 1);
  renderProgrammingWeekPicker();
  programmingWeekPicker.hidden = false;
  programmingWeekLabel.setAttribute("aria-expanded", "true");
  programmingWeekPicker.querySelector(".programming-picker-week.selected")?.focus();
}

function closeProgrammingWeekPicker() {
  if (!programmingWeekPicker) return;
  programmingWeekPicker.hidden = true;
  programmingWeekLabel.setAttribute("aria-expanded", "false");
}

function toggleProgrammingWeekPicker() {
  if (programmingWeekPicker.hidden) openProgrammingWeekPicker();
  else closeProgrammingWeekPicker();
}

function programmingCollectionKind(collection) {
  const types = new Set([
    ...(collection.rules || []).map((rule) => rule.media_type),
    ...(collection.pinned_media || []).map((item) => item.media_type),
  ].filter(Boolean));
  return types.size === 1 ? [...types][0] : "mixed";
}

function programmingCollectionArtwork(collection) {
  if (!collection?.artwork_path) return "none";
  return `url("${collection.artwork_path.replaceAll('"', "%22")}")`;
}

function programmingPosterArtwork(poster) {
  const match = String(poster || "").match(/url\((?:"[^"]*"|'[^']*'|[^)]*)\)/);
  return match?.[0] || "none";
}

function programmingBlockMedia(block) {
  const item = block?.media_items?.[0];
  if (!item) return null;
  const source = item.media_type === "tv" ? tvShows : movies;
  return source.find((media) => Number(media.id) === Number(item.media_id)) || null;
}

function programmingBlockArtwork(block, collection) {
  if (collection) return programmingCollectionArtwork(collection);
  return programmingPosterArtwork(programmingBlockMedia(block)?.poster);
}

function renderProgrammingDropMode() {
  const mode = document.querySelector('input[name="programming-schedule-mode"]:checked')?.value || "weekly";
  programmingDropWeekly.hidden = mode !== "weekly";
  programmingDropDateField.hidden = mode !== "once";
  programmingDropDays.querySelectorAll("[data-drop-day]").forEach((button) => button.classList.toggle("active", pendingProgrammingDropDays.includes(button.dataset.dropDay)));
  queueProgrammingDropScheduleCheck();
}

function programmingDropWindows() {
  const mode = document.querySelector('input[name="programming-schedule-mode"]:checked')?.value || "weekly";
  const dates = [];
  if (mode === "once" && programmingDropDate.value) {
    dates.push(programmingDropDate.value);
  } else if (mode === "weekly") {
    const week = programmingWeekStart();
    ["mon", "tue", "wed", "thu", "fri", "sat", "sun"].forEach((day, index) => {
      if (!pendingProgrammingDropDays.includes(day)) return;
      const date = new Date(week); date.setDate(date.getDate() + index);
      dates.push(localDateKey(date));
    });
  }
  if (!programmingDropStart.value || !programmingDropEnd.value) return [];
  return dates.map((dateKey) => {
    const start = new Date(`${dateKey}T${programmingDropStart.value}:00`);
    const end = new Date(`${dateKey}T${programmingDropEnd.value}:00`);
    if (Number.isNaN(start.getTime()) || Number.isNaN(end.getTime())) return null;
    if (end <= start) end.setDate(end.getDate() + 1);
    return { start, end };
  }).filter(Boolean);
}

function queueProgrammingDropScheduleCheck() {
  window.clearTimeout(programmingDropScheduleCheckTimer);
  programmingDropScheduleCheckTimer = window.setTimeout(checkProgrammingDropSchedule, 180);
}

async function checkProgrammingDropSchedule() {
  const channel = selectedProgrammingChannel();
  const windows = programmingDropWindows();
  const sequence = ++programmingDropScheduleCheckSequence;
  if (!pendingProgrammingDrop || !channel || windows.length === 0) {
    programmingScheduleNotice.hidden = true;
    programmingDropConfirm.textContent = "Add to schedule";
    return;
  }
  try {
    const counts = await Promise.all(windows.map(async (window) => {
      const params = new URLSearchParams({ start: window.start.toISOString(), end: window.end.toISOString(), summary: "1" });
      const response = await fetch(`/api/channels/${channel.id}/schedule?${params}`);
      if (!response.ok) throw new Error("Could not check generated schedule");
      return Number((await response.json()).count || 0);
    }));
    if (sequence !== programmingDropScheduleCheckSequence || !pendingProgrammingDrop) return;
    const overlaps = counts.some((count) => count > 0);
    programmingScheduleNotice.hidden = !overlaps;
    programmingDropConfirm.textContent = overlaps ? "Update Schedule" : "Add to schedule";
  } catch (_) {
    if (sequence !== programmingDropScheduleCheckSequence) return;
    programmingScheduleNotice.hidden = true;
    programmingDropConfirm.textContent = "Add to schedule";
  }
}

function openProgrammingDropModal(details) {
  if (details.media) {
    const source = details.media.media_type === "tv" ? tvShows : movies;
    const record = source.find((item) => Number(item.id) === Number(details.media.media_id));
    if (record) details.media = { ...details.media, ...record, media_id: details.media.media_id, media_type: details.media.media_type };
  }
  pendingProgrammingDrop = details;
  pendingProgrammingDropDays = [details.day];
  const isTV = details.media?.media_type === "tv";
  const isMovie = details.media?.media_type === "movie";
  programmingDropSource.textContent = details.collection ? `Collection • ${details.collection.name}` : `${isTV ? "TV show" : "Movie"} • ${details.media.title}`;
  programmingDropStart.value = details.startTime;
  if (isMovie && Number(details.media.runtimeMinutes) > 0) {
    const [startHour, startMinute] = details.startTime.split(":").map(Number);
    const scheduledDuration = Math.ceil(Number(details.media.runtimeMinutes) / 30) * 30;
    const endMinutes = startHour * 60 + startMinute + scheduledDuration;
    programmingDropEnd.value = `${String(Math.floor(endMinutes / 60) % 24).padStart(2, "0")}:${String(endMinutes % 60).padStart(2, "0")}`;
  } else {
    programmingDropEnd.value = details.endTime;
  }
  programmingDropDate.value = details.date;
  programmingDropPlayback.value = isTV ? "chronological" : details.collection ? "shuffle" : "in_order";
  programmingDropPlaybackField.hidden = isMovie;
  programmingDropError.textContent = "";
  const weekly = document.querySelector('input[name="programming-schedule-mode"][value="weekly"]');
  const once = document.querySelector('input[name="programming-schedule-mode"][value="once"]');
  if (isMovie && once) once.checked = true;
  else if (weekly) weekly.checked = true;
  renderProgrammingDropMode();
  programmingDropModal.hidden = false;
  programmingDropModal.classList.add("open");
  programmingDropModal.setAttribute("aria-hidden", "false");
  programmingDropModal.querySelector('input[name="programming-schedule-mode"]:checked')?.focus();
}

function closeProgrammingDropModal() {
  programmingDropModal.classList.remove("open");
  programmingDropModal.setAttribute("aria-hidden", "true");
  programmingDropModal.hidden = true;
  pendingProgrammingDrop = null;
  programmingDropScheduleCheckSequence += 1;
  window.clearTimeout(programmingDropScheduleCheckTimer);
  programmingScheduleNotice.hidden = true;
  programmingDropConfirm.textContent = "Add to schedule";
}

function programmingOccurrenceDate(day) {
  const dayIndex = ["mon", "tue", "wed", "thu", "fri", "sat", "sun"].indexOf(day);
  const date = programmingWeekStart();
  date.setDate(date.getDate() + Math.max(0, dayIndex));
  return localDateKey(date);
}

function closeProgrammingRecurrenceModal(restore = true) {
  programmingRecurrenceModal.classList.remove("open");
  programmingRecurrenceModal.setAttribute("aria-hidden", "true");
  programmingRecurrenceModal.hidden = true;
  pendingProgrammingRecurrenceAdjustment = null;
  if (restore) renderProgrammingCalendar();
}

function requestProgrammingRecurrenceScope(block, day, applyChange) {
  if ((block.schedule_mode || "weekly") !== "weekly") {
    applyChange(block, "all");
    return;
  }
  pendingProgrammingRecurrenceAdjustment = { block, day, applyChange };
  programmingRecurrenceModal.hidden = false;
  programmingRecurrenceModal.classList.add("open");
  programmingRecurrenceModal.setAttribute("aria-hidden", "false");
  programmingRecurrenceModal.querySelector("[data-recurrence-scope]")?.focus();
}

function applyProgrammingRecurrenceScope(scope) {
  const pending = pendingProgrammingRecurrenceAdjustment;
  if (!pending) return;
  let editable = pending.block;
  if (scope === "one") {
    const occurrenceDate = programmingOccurrenceDate(pending.day);
    editable.excluded_dates = [...new Set([...(editable.excluded_dates || []), occurrenceDate])];
    editable = createProgrammingBlock({ ...editable, id: null, days: [pending.day], schedule_mode: "once", exact_date: occurrenceDate, excluded_dates: [] });
    programmingPageBlocks.push(editable);
  }
  const applyChange = pending.applyChange;
  closeProgrammingRecurrenceModal(false);
  applyChange(editable, scope);
}

function confirmProgrammingDrop() {
  if (!pendingProgrammingDrop) return;
  const scheduleMode = document.querySelector('input[name="programming-schedule-mode"]:checked')?.value || "weekly";
  if (scheduleMode === "weekly" && pendingProgrammingDropDays.length === 0) { programmingDropError.textContent = "Choose at least one repeat day."; return; }
  if (scheduleMode === "once" && !programmingDropDate.value) { programmingDropError.textContent = "Choose the event date."; return; }
  if (!programmingDropStart.value || !programmingDropEnd.value || programmingDropStart.value === programmingDropEnd.value) { programmingDropError.textContent = "Choose different start and end times."; return; }
  const { collection, media, day } = pendingProgrammingDrop;
  const eventDay = scheduleMode === "once" ? ["sun", "mon", "tue", "wed", "thu", "fri", "sat"][new Date(`${programmingDropDate.value}T00:00:00`).getDay()] : day;
  const block = createProgrammingBlock({
    name: collection?.name || media.title, placement: "timeline",
    start_time: programmingDropStart.value, end_time: programmingDropEnd.value,
    days: scheduleMode === "weekly" ? pendingProgrammingDropDays : [eventDay],
    schedule_mode: scheduleMode, exact_date: scheduleMode === "once" ? programmingDropDate.value : "",
    playback_mode: programmingDropPlayback.value,
    content_source: collection ? "collection" : "media", collection_id: collection?.id || 0,
    rule_set: collection?.name || (media.media_type === "tv" ? "TV Show" : "Movie"),
    media_items: media ? [{ media_type: media.media_type, media_id: media.media_id, title: media.title }] : [],
  });
  programmingPageBlocks.push(block);
  programmingPageSelectedBlockID = block.id;
  closeProgrammingDropModal();
  renderProgrammingPage();
}

function dedupeProgrammingBlocks(blocks) {
  const seen = new Set();
  return blocks.filter((block) => {
    const signature = JSON.stringify([
      block.placement || "timeline",
      Number(block.collection_id || 0),
      block.name || block.title || "",
      block.start_time || "",
      block.end_time || "",
      block.schedule_mode || "weekly",
      block.exact_date || "",
      [...(block.excluded_dates || [])].sort(),
      [...(block.days || [])].sort(),
    ]);
    if (seen.has(signature)) return false;
    seen.add(signature);
    return true;
  });
}

function renderProgrammingPage() {
  if (!programmingChannelSelect) return;
  const selectedChannel = channels.find((channel) => Number(channel.id) === Number(programmingPageChannelID)) || channels[0] || null;
  if (selectedChannel && Number(programmingPageChannelID) !== Number(selectedChannel.id)) {
    programmingPageChannelID = selectedChannel.id;
    programmingPageBlocks = dedupeProgrammingBlocks(normalizeProgrammingItems(selectedChannel.programming_blocks, "block"));
    programmingPageSelectedBlockID = programmingPageBlocks[0]?.id || null;
  }
  programmingChannelSelect.innerHTML = channels.map((channel, index) => `<option value="${channel.id}">${String(index + 1).padStart(2, "0")}  ${escapeHTML(channel.name)}</option>`).join("");
  if (selectedChannel) programmingChannelSelect.value = String(selectedChannel.id);
  renderProgrammingCollections();
  renderProgrammingAllDayLane();
  renderProgrammingCalendar();
  renderProgrammingInspector();
}

function renderProgrammingCollections() {
  if (!programmingCollectionList) return;
  const query = programmingCollectionSearch.value.trim().toLowerCase();
  programmingSourceTabs.forEach((tab) => {
    const active = tab.dataset.programmingSource === programmingSourceTab;
    tab.classList.toggle("active", active);
    tab.setAttribute("aria-selected", String(active));
  });
  programmingCollectionSearch.placeholder = programmingSourceTab === "movies" ? "Search movies..." : programmingSourceTab === "tvshows" ? "Search TV shows..." : "Search collections...";
  programmingCollectionType.hidden = programmingSourceTab !== "collections";
  programmingNewCollection.hidden = programmingSourceTab !== "collections";
  if (programmingSourceTab !== "collections") {
    const source = programmingSourceTab === "movies"
      ? movies.map((movie) => ({ media_type: "movie", media_id: movie.id, title: movie.title, poster: movie.poster, subtitle: [movie.year, movie.runtime].filter(Boolean).join(" • ") || "Movie" }))
      : tvShows.map((show) => ({ media_type: "tv", media_id: show.id, title: show.title, poster: show.poster, subtitle: [show.year, show.network].filter(Boolean).join(" • ") || "TV Show" }));
    const visibleMedia = source.filter((item) => !query || item.title.toLowerCase().includes(query));
    programmingCollectionList.innerHTML = visibleMedia.length ? visibleMedia.map((item) => `<button class="programming-collection-row programming-media-row" type="button" draggable="true" data-programming-media-type="${item.media_type}" data-programming-media-id="${item.media_id}"><span class="programming-collection-art" style="--art:${escapeAttribute(item.poster || "linear-gradient(145deg,#24334a,#080d14)")}"></span><span><strong>${escapeHTML(item.title)}</strong><small>${escapeHTML(item.subtitle)}</small></span><span>⠿</span></button>`).join("") : `<p class="programming-empty">No ${programmingSourceTab === "movies" ? "movies" : "TV shows"} found.</p>`;
    programmingCollectionList.querySelectorAll("[data-programming-media-id]").forEach((row) => {
      row.addEventListener("dragstart", (event) => event.dataTransfer.setData("text/programming-media", JSON.stringify({ media_type: row.dataset.programmingMediaType, media_id: Number(row.dataset.programmingMediaId), title: row.querySelector("strong")?.textContent || "Media" })));
    });
    return;
  }
  const type = programmingCollectionType.value;
  const visible = collections.filter((collection) => String(collection.name || "").toLowerCase().includes(query)).filter((collection) => !type || programmingCollectionKind(collection) === type);
  programmingCollectionList.innerHTML = visible.length ? visible.map((collection, index) => {
    const count = (collection.pinned_media || []).length;
    const art = collection.artwork_path ? `url(&quot;${escapeAttribute(collection.artwork_path)}&quot;) center / contain no-repeat` : `linear-gradient(145deg,hsl(${(index * 47 + 275) % 360} 62% 38%),#07101b)`;
    return `<button class="programming-collection-row" type="button" draggable="true" data-programming-collection="${collection.id}"><span class="programming-collection-art" style="--art:${art}"></span><span><strong>${escapeHTML(collection.name)}</strong><small>${count} pinned items • ${programmingCollectionKind(collection)}</small></span><span>⠿</span></button>`;
  }).join("") : `<p class="programming-empty">No collections found.</p>`;
  programmingCollectionList.querySelectorAll("[data-programming-collection]").forEach((row) => {
    row.addEventListener("dragstart", (event) => event.dataTransfer.setData("text/programming-collection", row.dataset.programmingCollection));
  });
}

function selectedProgrammingChannel() {
  return channels.find((channel) => Number(channel.id) === Number(programmingPageChannelID)) || null;
}

function renderProgrammingAllDayLane() {
  if (!programmingAllDayLane) return;
  const dayKeys = ["mon", "tue", "wed", "thu", "fri", "sat", "sun"];
  const week = programmingWeekStart();
  const blocks = programmingPageBlocks.filter((block) => block.placement === "all_day").sort((left, right) => Number(right.priority || 0) - Number(left.priority || 0));
  programmingAllDayLane.innerHTML = `<span class="programming-week-corner"></span>${dayKeys.map((day, index) => { const date = new Date(week); date.setDate(date.getDate() + index); return `<div class="programming-week-day${localDateKey(date) === localDateKey(new Date()) ? " active" : ""}"><strong>${date.toLocaleDateString([], { weekday: "short" })}</strong><span>${date.toLocaleDateString([], { month: "numeric", day: "numeric" })}</span></div>`; }).join("")}<div class="programming-all-day-title">ALL DAY</div>${blocks.length ? blocks.map((block, index) => { const indexes = (block.days || []).map((day) => dayKeys.indexOf(String(day).slice(0, 3).toLowerCase())).filter((day) => day >= 0); const start = indexes.length ? Math.min(...indexes) : 0; const end = indexes.length ? Math.max(...indexes) : 6; return `<button class="programming-all-day-block${block.id === programmingPageSelectedBlockID ? " selected" : ""}" style="grid-column:${start + 2}/${end + 3};grid-row:${index + 2}" type="button" draggable="true" data-all-day-block="${escapeAttribute(block.id)}"><span class="all-day-resize all-day-resize-start" data-all-day-edge="start" aria-hidden="true"></span><span class="programming-all-day-copy"><strong>${escapeHTML(block.name)} <em>(All Day Fallback)</em></strong><small>Collection: ${escapeHTML(block.rule_set || "Collection")}</small></span><span class="programming-all-day-grip">⠿</span><span class="all-day-resize all-day-resize-end" data-all-day-edge="end" aria-hidden="true"></span></button>`; }).join("") : `<p class="programming-all-day-empty">Drop a collection here to set fallback programming</p>`}`;
  programmingAllDayLane.querySelectorAll("[data-all-day-block]").forEach((row) => {
    const block = programmingPageBlocks.find((item) => String(item.id) === row.dataset.allDayBlock);
    row.addEventListener("click", () => { programmingPageSelectedBlockID = block.id; renderProgrammingPage(); });
    row.addEventListener("dragstart", (event) => {
      if (event.target.closest("[data-all-day-edge]")) { event.preventDefault(); return; }
      event.dataTransfer.setData("text/programming-block", JSON.stringify({ id: block.id, day: block.days?.[0] || "mon", allDay: true }));
    });
    row.querySelectorAll("[data-all-day-edge]").forEach((handle) => handle.addEventListener("pointerdown", (event) => {
      event.preventDefault(); event.stopPropagation();
      const indexes = (block.days || []).map((day) => dayKeys.indexOf(day)).filter((day) => day >= 0);
      let start = indexes.length ? Math.min(...indexes) : 0;
      let end = indexes.length ? Math.max(...indexes) : 6;
      const initialStart = start; const initialEnd = end; const startX = event.clientX;
      const dayWidth = Math.max(1, (programmingAllDayLane.getBoundingClientRect().width - 64) / 7);
      const move = (moveEvent) => {
        const delta = Math.round((moveEvent.clientX - startX) / dayWidth);
        if (handle.dataset.allDayEdge === "start") start = Math.max(0, Math.min(initialEnd, initialStart + delta));
        else end = Math.min(6, Math.max(initialStart, initialEnd + delta));
        row.style.gridColumn = `${start + 2}/${end + 3}`;
      };
      const endResize = () => {
        document.removeEventListener("pointermove", move);
        document.removeEventListener("pointerup", endResize);
        block.days = dayKeys.slice(start, end + 1);
        programmingPageSelectedBlockID = block.id;
        renderProgrammingPage();
      };
      document.addEventListener("pointermove", move);
      document.addEventListener("pointerup", endResize, { once: true });
    }));
  });
  programmingAllDayLane.ondragover = (event) => event.preventDefault();
  programmingAllDayLane.ondrop = (event) => {
    event.preventDefault();
    const movingData = event.dataTransfer.getData("text/programming-block");
    if (movingData) {
      try {
        const moving = JSON.parse(movingData);
        const block = programmingPageBlocks.find((item) => String(item.id) === String(moving.id));
        if (!block) return;
        block.placement = "all_day";
        block.start_time = "";
        block.end_time = "";
        block.days = ["mon", "tue", "wed", "thu", "fri", "sat", "sun"];
        const rect = programmingAllDayLane.getBoundingClientRect();
        const targetDay = Math.max(0, Math.min(6, Math.floor(((event.clientX - rect.left - 64) / Math.max(1, rect.width - 64)) * 7)));
        const oldIndexes = (block.days || []).map((day) => dayKeys.indexOf(day)).filter((day) => day >= 0);
        const span = oldIndexes.length ? Math.max(...oldIndexes) - Math.min(...oldIndexes) : 0;
        const newStart = Math.min(targetDay, 6 - span);
        block.days = dayKeys.slice(newStart, newStart + span + 1);
        programmingPageSelectedBlockID = block.id;
        renderProgrammingPage();
      } catch (_) {}
      return;
    }
    const collection = collections.find((item) => String(item.id) === event.dataTransfer.getData("text/programming-collection"));
    if (!collection) return;
    const rect = programmingAllDayLane.getBoundingClientRect();
    const targetDay = Math.max(0, Math.min(6, Math.floor(((event.clientX - rect.left - 64) / Math.max(1, rect.width - 64)) * 7)));
    const block = createProgrammingBlock({ name: collection.name, placement: "all_day", start_time: "", end_time: "", days: [dayKeys[targetDay]], content_source: "collection", collection_id: collection.id, rule_set: collection.name, priority: blocks.length });
    programmingPageBlocks.push(block);
    programmingPageSelectedBlockID = block.id;
    renderProgrammingPage();
  };
}

async function publishProgrammingPage() {
  const channel = selectedProgrammingChannel();
  if (!channel) return;
  programmingPublish.disabled = true;
  programmingPublish.textContent = "Publishing…";
  try {
    const publishableBlocks = programmingPageBlocks.map(({ id, type, active, ...block }) => ({
      ...block,
      id: Number.isInteger(id) ? id : 0,
    }));
    const response = await fetch(`/api/channels/${channel.id}/programming`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ programming_blocks: publishableBlocks }),
    });
    const saved = await readJSONResponse(response);
    if (!response.ok) throw new Error(saved.error || "Could not publish programming");
    upsertChannel(saved);
    programmingPageBlocks = dedupeProgrammingBlocks(normalizeProgrammingItems(saved.programming_blocks, "block"));
    programmingPageSelectedBlockID = programmingPageBlocks[0]?.id || null;
    programmingPublish.textContent = "Refreshing guide…";
    renderProgrammingPage();
    guideDayCache.clear();
    await loadGuide(guideDateValue, false);
    programmingPublish.textContent = "Published";
  } catch (error) {
    programmingPublish.textContent = error.message;
  } finally {
    window.setTimeout(() => {
      programmingPublish.disabled = false;
      programmingPublish.textContent = "Publish Changes";
    }, 1600);
  }
}

function renderProgrammingCalendar() {
  if (!programmingCalendar) return;
  const week = programmingWeekStart();
  const end = new Date(week); end.setDate(end.getDate() + 6);
  programmingWeekLabel.textContent = `${week.toLocaleDateString([], { month: "short", day: "numeric" })} – ${end.toLocaleDateString([], { month: "short", day: "numeric", year: "numeric" })}`;
  const dayKeys = ["mon", "tue", "wed", "thu", "fri", "sat", "sun"];
  const grid = document.createElement("div"); grid.className = "programming-calendar-grid";
  dayKeys.forEach((day, dayIndex) => {
    const column = document.createElement("span");
    column.className = "programming-day-column";
    column.dataset.day = day;
    column.style.gridColumn = String(dayIndex + 2);
    column.style.gridRow = "1 / 49";
    grid.append(column);
  });
  for (let slot = 0; slot < 48; slot += 1) {
    const line = document.createElement("span");
    line.className = `programming-time-line ${slot % 2 === 0 ? "hour" : "half-hour"}`;
    line.style.gridColumn = "2 / 9";
    line.style.gridRow = String(slot + 1);
    grid.append(line);
  }
  for (let hour = 1; hour < 24; hour += 1) {
    const label = document.createElement("div"); label.className = "programming-time";
    label.textContent = new Date(2000, 0, 1, hour).toLocaleTimeString([], { hour: "numeric" }); label.style.gridColumn = "1"; label.style.gridRow = String(hour * 2 + 1); grid.append(label);
  }
  programmingPageBlocks.filter((block) => block.placement !== "all_day").forEach((block) => {
    const [hour, minute] = String(block.start_time || "00:00").split(":").map(Number);
    const [endHour, endMinute] = String(block.end_time || "02:00").split(":").map(Number);
    const duration = Math.max(1, ((endHour * 60 + endMinute) - (hour * 60 + minute) + 1440) % 1440 || 120);
    (block.days || []).forEach((day) => {
      const dayIndex = dayKeys.indexOf(String(day).slice(0, 3).toLowerCase()); if (dayIndex < 0) return;
      const occurrenceDate = new Date(week); occurrenceDate.setDate(occurrenceDate.getDate() + dayIndex);
      if ((block.excluded_dates || []).includes(localDateKey(occurrenceDate))) return;
      if (block.schedule_mode === "once") {
        if (localDateKey(occurrenceDate) !== block.exact_date) return;
      }
      const chip = document.createElement("button");
      const collection = collections.find((item) => Number(item.id) === Number(block.collection_id));
      const mediaType = block.media_items?.[0]?.media_type;
      const kind = collection ? programmingCollectionKind(collection) : (mediaType === "tv" ? "tv" : mediaType === "movie" ? "movie" : "mixed");
      chip.className = `programming-block-chip ${kind}${block.id === programmingPageSelectedBlockID ? " selected" : ""}`;
      chip.style.setProperty("--block-artwork", programmingBlockArtwork(block, collection));
      chip.draggable = true;
      chip.dataset.blockId = block.id;
      chip.dataset.blockDay = day;
      chip.style.gridColumn = String(dayIndex + 2); chip.style.gridRow = `${Math.floor((hour * 60 + minute) / 30) + 1} / span ${Math.max(1, Math.ceil(duration / 30))}`;
      chip.innerHTML = `<strong>${escapeHTML(block.name)}</strong><small>${duration >= 60 ? `${Math.round(duration / 6) / 10}h` : `${duration}m`}</small><span class="programming-block-menu-trigger" role="button" tabindex="0" aria-label="Open block menu" title="Block options">•••</span><span class="programming-resize-handle" title="Drag to resize" aria-hidden="true"></span>`;
      chip.addEventListener("click", () => { programmingPageSelectedBlockID = block.id; renderProgrammingCalendar(); renderProgrammingInspector(); });
      const menuTrigger = chip.querySelector(".programming-block-menu-trigger");
      const openMenu = (event) => {
        event.preventDefault();
        event.stopPropagation();
        openProgrammingBlockContextMenu(event, block, day);
      };
      menuTrigger.addEventListener("click", openMenu);
      menuTrigger.addEventListener("pointerdown", (event) => event.stopPropagation());
      menuTrigger.addEventListener("keydown", (event) => {
        if (event.key === "Enter" || event.key === " ") openMenu(event);
      });
      chip.addEventListener("contextmenu", openMenu);
      chip.addEventListener("dragstart", (event) => {
        if (event.target.closest(".programming-resize-handle, .programming-block-menu-trigger")) { event.preventDefault(); return; }
        event.dataTransfer.effectAllowed = "move";
        const chipRect = chip.getBoundingClientRect();
        const rowHeight = Math.max(1, grid.getBoundingClientRect().height / 48);
        const grabOffsetSlots = Math.max(0, Math.round((event.clientY - chipRect.top) / rowHeight));
        event.dataTransfer.setData("text/programming-block", JSON.stringify({ id: block.id, day, grabOffsetSlots }));
        chip.classList.add("dragging");
      });
      chip.addEventListener("dragend", () => chip.classList.remove("dragging"));
      chip.querySelector(".programming-resize-handle").addEventListener("pointerdown", (event) => {
        event.preventDefault(); event.stopPropagation();
        const startY = event.clientY;
        const startingDuration = duration;
        chip.classList.add("resizing");
        const move = (moveEvent) => {
          const rowHeight = Math.max(1, grid.getBoundingClientRect().height / 48);
          const minutes = Math.round((startingDuration + ((moveEvent.clientY - startY) / rowHeight) * 30) / 30) * 30;
          const nextDuration = Math.max(30, Math.min(24 * 60 - (hour * 60 + minute), minutes));
          chip.style.gridRowEnd = `span ${Math.max(1, nextDuration / 30)}`;
          chip.querySelector("small").textContent = nextDuration >= 60 ? `${Math.round(nextDuration / 6) / 10}h` : `${nextDuration}m`;
        };
        const endResize = (upEvent) => {
          document.removeEventListener("pointermove", move);
          document.removeEventListener("pointerup", endResize);
          const rowHeight = Math.max(1, grid.getBoundingClientRect().height / 48);
          const minutes = Math.round((startingDuration + ((upEvent.clientY - startY) / rowHeight) * 30) / 30) * 30;
          const nextDuration = Math.max(30, Math.min(24 * 60 - (hour * 60 + minute), minutes));
          const endMinutes = hour * 60 + minute + nextDuration;
          const nextEndTime = `${String(Math.floor(endMinutes / 60) % 24).padStart(2, "0")}:${String(endMinutes % 60).padStart(2, "0")}`;
          requestProgrammingRecurrenceScope(block, day, (editable) => {
            editable.end_time = nextEndTime;
            programmingPageSelectedBlockID = editable.id;
            renderProgrammingPage();
          });
        };
        document.addEventListener("pointermove", move);
        document.addEventListener("pointerup", endResize, { once: true });
      });
      grid.append(chip);
    });
  });
  grid.addEventListener("dragover", (event) => event.preventDefault());
  grid.addEventListener("drop", (event) => {
    event.preventDefault();
    const rect = grid.getBoundingClientRect();
    const dayIndex = Math.max(0, Math.min(6, Math.floor(((event.clientX - rect.left - 64) / (rect.width - 64)) * 7)));
    const movingData = event.dataTransfer.getData("text/programming-block");
    if (movingData) {
      try {
        const moving = JSON.parse(movingData);
        const pointerSlot = Math.round(((event.clientY - rect.top) / rect.height) * 48);
        const slot = Math.max(0, Math.min(47, pointerSlot - Number(moving.grabOffsetSlots || 0)));
        const startMinutes = slot * 30;
        const startTime = `${String(Math.floor(startMinutes / 60)).padStart(2, "0")}:${String(startMinutes % 60).padStart(2, "0")}`;
        const block = programmingPageBlocks.find((item) => String(item.id) === String(moving.id));
        if (!block) return;
        const [oldStartHour, oldStartMinute] = String(block.start_time).split(":").map(Number);
        const [oldEndHour, oldEndMinute] = String(block.end_time).split(":").map(Number);
        const duration = Math.max(30, ((oldEndHour * 60 + oldEndMinute) - (oldStartHour * 60 + oldStartMinute) + 1440) % 1440 || 120);
        const endMinutes = Math.min(1440, startMinutes + duration);
        const targetDay = dayKeys[dayIndex];
        const targetDate = new Date(week); targetDate.setDate(targetDate.getDate() + dayIndex);
        requestProgrammingRecurrenceScope(block, moving.day, (editable, scope) => {
          editable.placement = "timeline";
          if (scope === "one" || editable.schedule_mode === "once") {
            editable.days = [targetDay];
            editable.exact_date = localDateKey(targetDate);
          } else {
            const sourceIndex = dayKeys.indexOf(moving.day);
            const offset = dayIndex - sourceIndex;
            editable.days = [...new Set((editable.days || []).map((scheduledDay) => dayKeys[(dayKeys.indexOf(scheduledDay) + offset + 7) % 7]))];
          }
          editable.start_time = startTime;
          editable.end_time = `${String(Math.floor(endMinutes / 60) % 24).padStart(2, "0")}:${String(endMinutes % 60).padStart(2, "0")}`;
          programmingPageSelectedBlockID = editable.id;
          renderProgrammingPage();
        });
      } catch (_) {}
      return;
    }
    const slot = Math.max(0, Math.min(47, Math.round(((event.clientY - rect.top) / rect.height) * 48)));
    const startMinutes = slot * 30;
    const startTime = `${String(Math.floor(startMinutes / 60)).padStart(2, "0")}:${String(startMinutes % 60).padStart(2, "0")}`;
    const collection = collections.find((item) => String(item.id) === event.dataTransfer.getData("text/programming-collection"));
    let media = null;
    if (!collection) {
      try { media = JSON.parse(event.dataTransfer.getData("text/programming-media") || "null"); } catch (_) {}
      if (!media?.media_id || !["movie", "tv"].includes(media.media_type)) return;
    }
    const endMinutes = Math.min(1440, startMinutes + 120);
    const targetDate = new Date(week); targetDate.setDate(targetDate.getDate() + dayIndex);
    openProgrammingDropModal({
      collection, media, day: dayKeys[dayIndex], date: localDateKey(targetDate), startTime,
      endTime: `${String(Math.floor(endMinutes / 60) % 24).padStart(2, "0")}:${String(endMinutes % 60).padStart(2, "0")}`,
    });
  });
  programmingCalendar.replaceChildren(grid);
}

function closeProgrammingBlockContextMenu() {
  document.querySelector(".programming-block-context-menu")?.remove();
}

function openProgrammingBlockContextMenu(event, block, day) {
  closeProgrammingBlockContextMenu();
  programmingPageSelectedBlockID = block.id;
  const menu = document.createElement("div");
  menu.className = "programming-block-context-menu";
  menu.setAttribute("role", "menu");
  menu.addEventListener("pointerdown", (pointerEvent) => pointerEvent.stopPropagation());
  const recurring = (block.days || []).length > 1;
  menu.innerHTML = `<button type="button" data-block-action="duplicate" role="menuitem">Duplicate occurrence</button>${recurring ? `<button type="button" data-block-action="remove-occurrence" role="menuitem">Remove this occurrence</button>` : ""}<button class="danger" type="button" data-block-action="remove-block" role="menuitem">Remove ${recurring ? "entire block" : "block"}</button>`;
  document.body.append(menu);
  const rect = event.currentTarget?.getBoundingClientRect?.();
  const anchorX = event.clientX || rect?.right || 0;
  const anchorY = event.clientY || rect?.bottom || 0;
  const menuRect = menu.getBoundingClientRect();
  menu.style.left = `${Math.max(8, Math.min(window.innerWidth - menuRect.width - 8, anchorX))}px`;
  menu.style.top = `${Math.max(8, Math.min(window.innerHeight - menuRect.height - 8, anchorY))}px`;
  menu.querySelectorAll("[data-block-action]").forEach((button) => button.addEventListener("click", () => {
    switch (button.dataset.blockAction) {
    case "duplicate": {
      const duplicate = createProgrammingBlock({ ...block, id: null, days: [day], priority: Number(block.priority || 0) + 1 });
      programmingPageBlocks.push(duplicate);
      programmingPageSelectedBlockID = duplicate.id;
      break;
    }
    case "remove-occurrence":
      block.days = (block.days || []).filter((existingDay) => existingDay !== day);
      programmingPageSelectedBlockID = block.id;
      break;
    case "remove-block":
      programmingPageBlocks = programmingPageBlocks.filter((item) => item.id !== block.id);
      programmingPageSelectedBlockID = programmingPageBlocks[0]?.id || null;
      break;
    }
    closeProgrammingBlockContextMenu();
    renderProgrammingPage();
  }));
  window.setTimeout(() => document.addEventListener("pointerdown", closeProgrammingBlockContextMenu, { once: true }), 0);
  menu.querySelector("button")?.focus();
}

function snapProgrammingTime(value) {
  const [hour, minute] = String(value || "00:00").split(":").map(Number);
  const total = Math.max(0, Math.min(1439, (Number(hour) || 0) * 60 + (Number(minute) || 0)));
  const snapped = Math.min(1410, Math.round(total / 30) * 30);
  return `${String(Math.floor(snapped / 60)).padStart(2, "0")}:${String(snapped % 60).padStart(2, "0")}`;
}

function renderProgrammingInspector() {
  if (!programmingInspector) return;
  const block = programmingPageBlocks.find((item) => item.id === programmingPageSelectedBlockID);
  if (!block) { programmingInspector.innerHTML = `<div class="programming-inspector-empty">Select a schedule block to edit its placement.</div>`; return; }
  const timeFields = block.placement === "all_day" ? `<p class="programming-all-day-note">This collection fills every unscheduled part of the selected days.</p>` : `<div class="programming-inspector-row"><label>Starts<input class="inspector-start" type="time" step="1800" value="${escapeAttribute(block.start_time)}"></label><label>Ends<input class="inspector-end" type="time" step="1800" value="${escapeAttribute(block.end_time)}"></label></div>`;
  programmingInspector.innerHTML = `<header class="programming-inspector-head"><span class="programming-inspector-art"></span><div><h2>${escapeHTML(block.name)}</h2><p>${escapeHTML(block.rule_set || "Scheduled collection")}</p></div></header><section class="programming-inspector-section"><h3>${block.placement === "all_day" ? "All Day Programming" : "Timeline Placement"}</h3>${timeFields}</section><section class="programming-inspector-section"><h3>Repeat</h3><label>Days<input class="inspector-days" value="${escapeAttribute((block.days || []).join(", "))}"></label><label>Priority<input class="inspector-priority" type="number" value="${Number(block.priority || 0)}"></label></section><button class="button ghost programming-remove" type="button">Remove Block</button>`;
  const update = () => { const start = programmingInspector.querySelector(".inspector-start"); const end = programmingInspector.querySelector(".inspector-end"); if (start) { block.start_time = snapProgrammingTime(start.value); start.value = block.start_time; } if (end) { block.end_time = snapProgrammingTime(end.value); end.value = block.end_time; } block.days = programmingInspector.querySelector(".inspector-days").value.split(",").map((day) => day.trim().slice(0, 3).toLowerCase()).filter(Boolean); block.priority = Number(programmingInspector.querySelector(".inspector-priority").value || 0); renderProgrammingAllDayLane(); renderProgrammingCalendar(); };
  programmingInspector.querySelectorAll("input").forEach((input) => input.addEventListener("change", update));
  programmingInspector.querySelector(".programming-remove").addEventListener("click", () => { programmingPageBlocks = programmingPageBlocks.filter((item) => item.id !== block.id); programmingPageSelectedBlockID = programmingPageBlocks[0]?.id || null; renderProgrammingPage(); });
}

function render() {
  const query = search.value.trim().toLowerCase();
  const genre = movieGenreFilter?.value || "";
  const year = movieYearFilter?.value || "";
  const rating = movieRatingFilter?.value || "";
  const runtime = movieRuntimeFilter?.value || "";
  const visibleMovies = movies
    .filter((movie) => `${movie.title || ""} ${movie.year || ""}`.toLowerCase().includes(query))
    .filter((movie) => !genre || movie.genres.includes(genre))
    .filter((movie) => !year || String(movie.year) === year)
    .filter((movie) => !rating || movie.rating === rating)
    .filter((movie) => movieMatchesRuntime(movie, runtime))
    .filter(movieMatchesLibraryMode)
    .sort(compareMovies);

  movieCount.textContent = visibleMovies.length.toLocaleString();
  updateLibraryStats();

  if (visibleMovies.length === 0) {
    renderEmptyState(query);
    renderPagination(moviePagination, 1, 0, () => {});
    selectedMovie = null;
    renderDetails(null);
    return;
  }

  const totalPages = pageCount(visibleMovies.length);
  moviePage = clampPage(moviePage, totalPages);
  const pageMovies = pageItems(visibleMovies, moviePage);

  if (!pageMovies.includes(selectedMovie)) {
    selectedMovie = pageMovies[0];
  }

  try {
    grid.replaceChildren(...pageMovies.map(renderMovieCard));
    renderPagination(moviePagination, moviePage, totalPages, (page) => {
      moviePage = page;
      render();
    });
    renderDetails(selectedMovie);
  } catch (error) {
    scanStatus.textContent = `Could not render movies: ${error.message}`;
    renderEmptyState(query);
    renderPagination(moviePagination, 1, 0, () => {});
  }
}

function movieMatchesRuntime(movie, runtime) {
  if (!runtime) return true;
  const minutes = Number(movie.runtimeMinutes || 0);
  if (!minutes) return false;
  if (runtime === "short") return minutes < 90;
  if (runtime === "medium") return minutes >= 90 && minutes <= 120;
  return minutes > 120;
}

function movieMatchesLibraryMode(movie) {
  if (movieLibraryMode === "recent") {
    const added = new Date(movie.added).getTime();
    return Number.isFinite(added) && added >= Date.now() - 30 * 24 * 60 * 60 * 1000;
  }
  if (movieLibraryMode === "unmatched") return !movie.tmdbID || movie.missing;
  if (movieLibraryMode === "duplicates") return movie.duplicate;
  if (movieLibraryMode === "collections") return false;
  return true;
}

function compareMovies(left, right) {
  const titleOrder = String(left.title).localeCompare(String(right.title), undefined, { sensitivity: "base" });
  if (movieSortMode === "title-desc") return -titleOrder;
  if (movieSortMode === "year-desc") return Number(right.year || 0) - Number(left.year || 0) || titleOrder;
  if (movieSortMode === "year-asc") return Number(left.year || 0) - Number(right.year || 0) || titleOrder;
  if (movieSortMode === "added-desc") return new Date(right.added || 0) - new Date(left.added || 0) || titleOrder;
  return titleOrder;
}

function updateMovieFilterOptions() {
  replaceFilterOptions(movieGenreFilter, "All genres", movies.flatMap((movie) => movie.genres));
  replaceFilterOptions(movieYearFilter, "All years", movies.map((movie) => movie.year), (a, b) => Number(b) - Number(a));
  replaceFilterOptions(movieRatingFilter, "All ratings", movies.map((movie) => movie.rating));
}

function updateTVFilterOptions() {
  replaceFilterOptions(tvGenreFilter, "All genres", tvShows.flatMap((show) => show.genres || []));
  replaceFilterOptions(tvYearFilter, "All years", tvShows.map((show) => show.year), (a, b) => Number(b) - Number(a));
  replaceFilterOptions(tvRatingFilter, "All ratings", tvShows.map((show) => show.rating));
  replaceFilterOptions(tvNetworkFilter, "All networks", tvShows.map((show) => show.network));
}

function replaceFilterOptions(select, emptyLabel, values, compare) {
  if (!select) return;
  const current = select.value;
  const unique = [...new Set(values.filter(Boolean).map(String))].sort(compare || ((a, b) => a.localeCompare(b)));
  select.replaceChildren(new Option(emptyLabel, ""), ...unique.map((value) => new Option(value, value)));
  select.value = unique.includes(current) ? current : "";
}

function updateLibraryStats() {
  const setText = (id, value) => {
    const target = document.querySelector(id);
    if (target) target.textContent = value.toLocaleString();
  };
  setText("#library-total-movies", movies.length);
  setText("#library-matched-movies", movies.filter((movie) => movie.tmdbID && !movie.missing).length);
  setText("#library-unmatched-movies", movies.filter((movie) => !movie.tmdbID || movie.missing).length);
  setText("#library-duplicate-movies", movies.filter((movie) => movie.duplicate).length);
}

function renderEmptyState(query) {
  const empty = document.createElement("section");
  empty.className = "empty-state";

  const title = document.createElement("h2");
  const modeLabels = {
    collections: "No collections yet",
    recent: "No recently added movies",
    unmatched: "No unmatched movies",
    duplicates: "No duplicate movies",
  };
  title.textContent = query ? "No matching movies" : modeLabels[movieLibraryMode] || "No scanned movies";

  const message = document.createElement("p");
  message.textContent = query
    ? "Clear the search field or adjust the active filters."
    : movieLibraryMode === "collections"
      ? "Collection management will appear here when collections are configured."
      : movieLibraryMode === "all"
        ? "Set your media directory, save settings, then scan the library."
        : "No movies currently match this library view.";

  empty.append(title, message);
  grid.replaceChildren(empty);
}

function renderTVShows() {
  const query = tvSearch.value.trim().toLowerCase();
  const genre = tvGenreFilter?.value || "";
  const year = tvYearFilter?.value || "";
  const rating = tvRatingFilter?.value || "";
  const network = tvNetworkFilter?.value || "";
  const visibleShows = tvShows
    .filter((show) => `${show.title} ${show.year}`.toLowerCase().includes(query))
    .filter((show) => !genre || (show.genres || []).includes(genre))
    .filter((show) => !year || String(show.year) === year)
    .filter((show) => !rating || show.rating === rating)
    .filter((show) => !network || show.network === network);

  tvCount.textContent = visibleShows.length.toLocaleString();

  if (visibleShows.length === 0) {
    tvShowDetailOpen = false;
    renderTVEmptyState(query);
    renderPagination(tvPagination, 1, 0, () => {});
    selectedTVShow = null;
    renderTVDetails(null);
    renderTVShowFullDetail(null);
    return;
  }

  const totalPages = pageCount(visibleShows.length);
  tvShowPage = clampPage(tvShowPage, totalPages);
  const pageShows = pageItems(visibleShows, tvShowPage);

  if (!pageShows.includes(selectedTVShow)) {
    selectedTVShow = pageShows[0];
  }

  tvGrid.replaceChildren(...pageShows.map(renderTVShowCard));
  renderPagination(tvPagination, tvShowPage, totalPages, (page) => {
    tvShowPage = page;
    renderTVShows();
  });
  if (tvShowDetailOpen && selectedTVShow) {
    renderTVShowFullDetail(selectedTVShow);
  } else {
    renderTVDetails(selectedTVShow);
    renderTVShowFullDetail(null);
  }
}

function renderTVEmptyState(query) {
  const empty = document.createElement("section");
  empty.className = "empty-state";

  const title = document.createElement("h2");
  title.textContent = query ? "No matching TV shows" : "No scanned TV shows";

  const message = document.createElement("p");
  message.textContent = query
    ? "Clear the search field or scan more TV media."
    : "TV show scanning is ready for the library view once TV metadata is stored.";

  empty.append(title, message);
  tvGrid.replaceChildren(empty);
}

function pageCount(totalItems) {
  return Math.max(1, Math.ceil(totalItems / libraryPageSize));
}

function clampPage(page, totalPages) {
  return Math.min(Math.max(page, 1), Math.max(totalPages, 1));
}

function pageItems(items, page) {
  const start = (page - 1) * libraryPageSize;
  return items.slice(start, start + libraryPageSize);
}

function paginationPages(currentPage, totalPages) {
  if (totalPages <= 7) {
    return Array.from({ length: totalPages }, (_, index) => index + 1);
  }

  const pages = [1];
  const start = Math.max(2, currentPage - 1);
  const end = Math.min(totalPages - 1, currentPage + 1);
  if (start > 2) {
    pages.push("start-gap");
  }
  for (let page = start; page <= end; page++) {
    pages.push(page);
  }
  if (end < totalPages - 1) {
    pages.push("end-gap");
  }
  pages.push(totalPages);
  return pages;
}

function renderPagination(container, currentPage, totalPages, onPage) {
  if (!container) {
    return;
  }
  if (totalPages <= 1) {
    container.replaceChildren();
    return;
  }

  const previous = paginationButton("Prev", currentPage - 1, onPage);
  previous.disabled = currentPage <= 1;
  const next = paginationButton("Next", currentPage + 1, onPage);
  next.disabled = currentPage >= totalPages;

  const pageControls = paginationPages(currentPage, totalPages).map((page) => {
    if (typeof page === "string") {
      const gap = document.createElement("span");
      gap.textContent = "...";
      return gap;
    }
    const button = paginationButton(String(page), page, onPage);
    button.classList.toggle("active", page === currentPage);
    button.setAttribute("aria-current", page === currentPage ? "page" : "false");
    return button;
  });

  container.replaceChildren(previous, ...pageControls, next);
}

function paginationButton(label, page, onPage) {
  const button = document.createElement("button");
  button.type = "button";
  button.textContent = label;
  button.addEventListener("click", () => onPage(page));
  return button;
}

function renderTVShowCard(show) {
  const card = document.createElement("article");
  card.className = `show-card${show === selectedTVShow ? " selected" : ""}${show.missing ? " missing" : ""}`;

  const poster = document.createElement("button");
  poster.className = "show-poster-button";
  poster.type = "button";
  poster.style.setProperty("--poster", show.poster);
  poster.addEventListener("click", () => {
    selectedTVShow = show;
    tvShowDetailOpen = false;
    renderTVShows();
  });

  if (show.missing) {
    const status = document.createElement("span");
    status.className = "tv-presence-badge missing";
    status.textContent = "Missing";
    poster.append(status);
  } else if (show.duplicate) {
    const status = document.createElement("span");
    status.className = "tv-presence-badge duplicate";
    status.textContent = "Possible Duplicate";
    poster.append(status);
  }

  const title = document.createElement("p");
  title.className = "movie-title";
  title.textContent = show.title;

  const year = document.createElement("p");
  year.className = "movie-year";
  year.textContent = show.year || "Unknown";

  card.append(poster, title, year);
  return card;
}

function renderTVDetails(show) {
  if (!show) {
    tvDetailsPanel.innerHTML = `
      <section class="details-empty">
        <h2>No TV show selected</h2>
        <p>Scanned TV shows saved in the database will appear here.</p>
      </section>
    `;
    return;
  }

  tvDetailsPanel.innerHTML = `
    ${show.missing ? `
      <section class="tv-reconciliation-warning">
        <strong>Media folder is missing</strong>
        <p>RetroTV last saw this show ${escapeHTML(formatDate(show.lastSeenAt))}. Its previous path was <code>${escapeHTML(show.path)}</code>.</p>
        ${show.renameCandidate ? `<p>A current record named <strong>${escapeHTML(show.renameCandidate.title)}</strong> may be the renamed replacement.</p>` : ""}
        <p>Removing this entry deletes only its RetroTV database record and scheduled entries. It never deletes media files.</p>
      </section>` : show.duplicate ? `
      <section class="tv-reconciliation-warning duplicate">
        <strong>Possible duplicate record</strong>
        <p>Another TV-show record has the same title or TMDB identity. Review the paths before removing either record.</p>
      </section>` : ""}
    <div class="tv-detail-head">
      <h2>${escapeHTML(show.title)}</h2>
      <div class="detail-actions">
        <button class="button primary compact view-tv-seasons-button" type="button">View Seasons</button>
        <button class="button compact refresh-tv-metadata-button" type="button">Refresh Metadata</button>
        <button class="button danger compact delete-tv-show" type="button">${show.missing ? "Remove Record" : "Delete"}</button>
        <button class="details-close" type="button" aria-label="Close details">×</button>
      </div>
    </div>
    <div class="show-detail-poster" style="--poster: ${escapeAttribute(show.poster)}"></div>
    <section class="details-section">
      <div class="detail-list">
        ${detailRow("Year", show.year || "Unknown")}
        ${detailRow("Seasons", show.seasons || "Unknown")}
        ${detailRow("Episodes", show.episodes || "Unknown")}
        ${detailRow("Genres", (show.genres || []).join(", ") || "Unknown")}
        ${detailRow("Network", show.network || "Unknown")}
        ${detailRow("Rating", show.rating || "Unknown")}
      </div>
    </section>
    <section class="details-section">
      <h3>Overview</h3>
      <p>${escapeHTML(show.overview || "Metadata is not available for this show.")}</p>
    </section>
  `;

  tvDetailsPanel.querySelector(".view-tv-seasons-button")?.addEventListener("click", () => {
    selectedTVSeason = firstSeasonNumber(show);
    tvShowDetailOpen = true;
    renderTVShows();
  });
  tvDetailsPanel.querySelector(".refresh-tv-metadata-button")?.addEventListener("click", () => refreshTVShowMetadata(show));
  tvDetailsPanel.querySelector(".delete-tv-show")?.addEventListener("click", () => deleteTVShow(show));
}

function renderTVShowFullDetail(show) {
  if (!tvShowFullDetail || !tvLibraryPane || !tvDetailsPanel) {
    return;
  }

  if (!show) {
    tvShowFullDetail.hidden = true;
    tvLibraryPane.hidden = false;
    tvDetailsPanel.hidden = false;
    tvShowFullDetail.replaceChildren();
    return;
  }

  tvLibraryPane.hidden = true;
  tvDetailsPanel.hidden = true;
  tvShowFullDetail.hidden = false;

  const seasons = tvShowSeasons(show);
  const selectedSeason =
    seasons.find((season) => season.number === selectedTVSeason) || seasons[0] || { number: 1, episodes: [] };
  selectedTVSeason = selectedSeason.number;
  const selectedEpisode = selectedSeason.episodes[0] || null;
  const summary = tvShowEpisodeSummary(seasons);
  const seasonLabel = selectedSeason.number > 0 ? `Season ${selectedSeason.number}` : "Specials";
  const yearRange = show.year ? `${show.year} - Present` : "Unknown years";
  const genres = (show.genres || []).join(", ") || "Unknown genre";

  tvShowFullDetail.innerHTML = `
    <div class="tv-detail-breadcrumb">
      <button class="tv-back-button" type="button">TV Shows</button>
      <span>›</span>
      <strong>${escapeHTML(show.title)}</strong>
      <span>›</span>
      <strong>${escapeHTML(seasonLabel)}</strong>
    </div>

    <div class="tv-detail-layout">
      <aside class="tv-show-profile">
        <section class="tv-detail-card tv-show-profile-card">
          <div class="tv-show-profile-main">
            <div class="tv-show-profile-poster" style="--poster: ${escapeAttribute(show.poster)}"></div>
            <div>
              <h2>${escapeHTML(show.title)}</h2>
              <p>${escapeHTML(yearRange)} <span>${escapeHTML(show.rating || "NR")}</span></p>
              <p>${escapeHTML(genres)}</p>
              <p>${escapeHTML(show.overview || "Metadata is not available for this show.")}</p>
              <div class="tv-show-profile-actions">
                <button class="button compact" type="button">Edit Show</button>
                <button class="button ghost compact" type="button">Refresh Metadata</button>
              </div>
            </div>
          </div>
          <div class="tv-season-list">
            <h3>Seasons</h3>
            ${seasons.map((season) => `
              <button class="tv-season-row${season.number === selectedSeason.number ? " active" : ""}" type="button" data-season="${season.number}">
                <span>${escapeHTML(season.number > 0 ? `Season ${season.number}` : "Specials")}</span>
                <strong>${season.episodes.length}</strong>
              </button>
            `).join("")}
          </div>
        </section>

        <section class="tv-detail-card tv-show-summary-card">
          <h3>Show Summary</h3>
          <p><span class="status-dot ok"></span>${summary.available} Episodes Available</p>
          <p><span class="status-dot warn"></span>0 Episodes Eligible (Not Scheduled)</p>
          <p><span class="status-dot bad"></span>0 Episodes Unmatched</p>
          <p><span class="status-dot neutral"></span>${summary.missing} Episodes Missing Files</p>
        </section>
      </aside>

      <section class="tv-detail-card tv-episode-table-card">
        <div class="tv-episode-table-head">
          <div>
            <h2>${escapeHTML(seasonLabel)}</h2>
            <span>${selectedSeason.episodes.length} Episodes</span>
          </div>
          <div class="tv-detail-mode-toggle">
            <button class="active" type="button">Table</button>
            <button type="button">Grid</button>
          </div>
        </div>
        <div class="tv-episode-table">
          <div class="tv-episode-row tv-episode-header">
            <span>#</span>
            <span>Episode</span>
            <span>Runtime</span>
            <span>Status</span>
            <span>Scheduled</span>
          </div>
          ${selectedSeason.episodes.map((episode) => `
            <button class="tv-episode-row${selectedEpisode && episode.path === selectedEpisode.path ? " active" : ""}" type="button" data-episode-path="${escapeAttribute(episode.path)}">
              <span>${episode.episode || "-"}</span>
              <span class="tv-episode-name">
                <span class="tv-episode-thumb" style="--poster: ${escapeAttribute(episode.thumbnail || show.poster)}"></span>
                <strong>${escapeHTML(episode.title)}</strong>
              </span>
              <span>${episode.runtime || 22} min</span>
              <span><span class="status-dot ok"></span>Available</span>
              <span>${episode.scheduledLabel}</span>
            </button>
          `).join("")}
        </div>
      </section>

      <aside class="tv-detail-card tv-episode-inspector">
        ${episodeInspectorHTML(show, selectedEpisode, selectedSeason)}
      </aside>
    </div>
  `;

  tvShowFullDetail.querySelector(".tv-back-button")?.addEventListener("click", () => {
    tvShowDetailOpen = false;
    renderTVShows();
  });
  tvShowFullDetail.querySelectorAll(".tv-season-row").forEach((button) => {
    button.addEventListener("click", () => {
      selectedTVSeason = Number(button.dataset.season);
      renderTVShowFullDetail(show);
    });
  });
  tvShowFullDetail.querySelectorAll(".tv-episode-row[data-episode-path]").forEach((button) => {
    button.addEventListener("click", () => {
      const episode = selectedSeason.episodes.find((item) => item.path === button.dataset.episodePath);
      if (!episode) {
        return;
      }
      tvShowFullDetail.querySelectorAll(".tv-episode-row.active").forEach((row) => row.classList.remove("active"));
      button.classList.add("active");
      const inspector = tvShowFullDetail.querySelector(".tv-episode-inspector");
      if (inspector) {
        inspector.innerHTML = episodeInspectorHTML(show, episode, selectedSeason);
      }
    });
  });
}

function episodeInspectorHTML(show, episode, season) {
  if (!episode) {
    return `
      <section class="details-empty">
        <h2>No episode selected</h2>
        <p>This season does not have scanned episode files.</p>
      </section>
    `;
  }
  const fileName = episode.path.split(/[\\/]/).pop() || episode.path;
  const seasonLabel = season.number > 0 ? `Season ${season.number}` : "Specials";
  const technical = mediaTechnicalSummary(episode.mediaMetadata);
  return `
    <div class="tv-inspector-art" style="--poster: ${escapeAttribute(episode.thumbnail || show.poster)}"></div>
    <p>Episode ${episode.episode || "-"}</p>
    <h2>${escapeHTML(episode.title)}</h2>
    <div class="tv-inspector-tags">
      <span><span class="status-dot ok"></span>Available</span>
      <span>${episode.runtime || 22} min</span>
      <span>${escapeHTML(show.rating || "NR")}</span>
    </div>
    <section>
      <h3>Description</h3>
      <p>${escapeHTML(episode.overview || show.overview || "Episode metadata is not available for this file.")}</p>
    </section>
    <section class="tv-inspector-meta">
      <h3>Metadata</h3>
      ${detailRow("Season", seasonLabel)}
      ${detailRow("Episode", episode.episode || "Unknown")}
      ${detailRow("Network", show.network || "Unknown")}
      ${detailRow("Aired", episode.airDate || show.year || "Unknown")}
    </section>
    <section>
      <h3>Files</h3>
      <div class="tv-file-pill">
        <span class="status-dot ok"></span>
        <span>${escapeHTML(fileName)}</span>
      </div>
    </section>
    <section class="tv-inspector-meta">
      <h3>Technical</h3>
      ${detailRow("File Size", technical.fileSize)}
      ${detailRow("Video", technical.video)}
      ${detailRow("Audio", technical.audio)}
      ${detailRow("Subtitles", technical.subtitles)}
      ${technical.error ? detailRow("Probe Error", technical.error) : ""}
    </section>
    <section class="tv-inspector-meta">
      <h3>Scheduling</h3>
      ${detailRow("Status", "Available")}
      ${detailRow("Next Airing", episode.scheduledLabel)}
    </section>
  `;
}

function tvShowEpisodeSummary(seasons) {
  const available = seasons.reduce((total, season) => total + season.episodes.length, 0);
  return { available, missing: 0 };
}

function firstSeasonNumber(show) {
  const seasons = tvShowSeasons(show);
  return seasons[0]?.number || 1;
}

function tvShowSeasons(show) {
  const bySeason = new Map();
  const paths = Array.isArray(show.episodePaths) ? show.episodePaths : [];
  const metadataByPath = new Map((show.episodeMetadata || []).map((episode) => [episode.path, episode]));
  paths.forEach((path, index) => {
    const parsed = parseEpisodePath(path, index, metadataByPath.get(path));
    const episodes = bySeason.get(parsed.season) || [];
    episodes.push(parsed);
    bySeason.set(parsed.season, episodes);
  });

  return [...bySeason.entries()]
    .sort(([left], [right]) => left - right)
    .map(([number, episodes]) => ({
      number,
      episodes: episodes.sort((left, right) => left.episode - right.episode || left.title.localeCompare(right.title)),
    }));
}

function parseEpisodePath(path, index, metadata = null) {
  const normalized = String(path || "");
  const filename = normalized.split(/[\\/]/).pop() || `Episode ${index + 1}`;
  const match = filename.match(/S(\d{1,2})E(\d{1,3})/i);
  const season = metadata?.season || (match ? Number(match[1]) : 1);
  const episode = metadata?.episode || (match ? Number(match[2]) : index + 1);
  const fileTitle = filename
    .replace(/\.[^.]+$/, "")
    .replace(/S\d{1,2}E\d{1,3}/i, "")
    .replace(/[._-]+/g, " ")
    .replace(/\s+/g, " ")
    .trim() || `Episode ${episode}`;
  return {
    season,
    episode,
    title: metadata?.name || fileTitle,
    overview: metadata?.overview || "",
    airDate: metadata?.air_date || "",
    runtime: metadata?.runtime_minutes || 22,
    thumbnail: episodeThumbnailStyle(metadata?.thumbnail_local_path),
    mediaMetadata: metadata?.mediaMetadata || metadata?.media_metadata || {},
    path: normalized,
    scheduledLabel: episode <= 8 ? `Ch. ${selectedTVShow?.id || ""}` : "-",
  };
}

function episodeThumbnailStyle(path) {
  const url = imageURL(path);
  return url ? `url("${url}") center / cover` : "";
}

function renderChannels() {
  channelCount.textContent = channels.length === 0 ? "No channels created" : `${channels.length} channels`;
  if (exportChannelsButton) {
    exportChannelsButton.disabled = channels.length === 0;
  }

  if (channels.length === 0) {
    const empty = document.createElement("section");
    empty.className = "channel-empty";
    empty.innerHTML = `<h2>No channels yet</h2><p>Create a channel to start building a RetroTV lineup.</p>`;
    channelRows.replaceChildren(empty);
    return;
  }

  channelRows.replaceChildren(...channels.map(renderChannelRow));
}

function exportChannels() {
  if (channels.length === 0) {
    return;
  }
  window.location.href = "/api/channels/export";
}

function renderChannelRow(channel) {
  const row = document.createElement("article");
  row.className = "channel-list-item";
  row.dataset.channelId = channel.id;

  row.innerHTML = `
    <button class="drag-handle channel-drag-handle" type="button" draggable="true" aria-label="Drag ${escapeAttribute(channel.name)} to reorder">
      <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M9 5h.01M15 5h.01M9 12h.01M15 12h.01M9 19h.01M15 19h.01" /></svg>
    </button>
    <div class="channel-name-cell">
      ${channelLogoMarkup(channel.name, channel.logo_path)}
      <strong>${escapeHTML(channel.name)}</strong>
    </div>
    <div class="channel-actions">
      <button class="air-toggle" type="button" role="switch" aria-checked="${channel.airing !== false}" aria-label="Toggle ${escapeAttribute(channel.name)} on air">
        <span class="air-toggle-track" aria-hidden="true"><span class="air-toggle-knob"></span></span>
        <span class="air-toggle-label">${channel.airing !== false ? "On Air" : "Off Air"}</span>
      </button>
      <button class="button ghost compact edit-channel" type="button" data-view="channel-builder">
        <svg viewBox="0 0 24 24" aria-hidden="true"><path d="m12 20 8-8-4-4-8 8-2 6 6-2Z" /><path d="m14 6 4 4" /></svg>
        Edit
      </button>
      <button class="icon-button delete-channel" type="button" aria-label="Delete channel">
        <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M3 6h18M8 6V4h8v2M6 6l1 15h10l1-15" /></svg>
      </button>
    </div>
  `;

  const handle = row.querySelector(".channel-drag-handle");
	const airToggle = row.querySelector(".air-toggle");
	airToggle.addEventListener("click", async () => {
		const wasAiring = airToggle.getAttribute("aria-checked") === "true";
		const requestedAiring = !wasAiring;
		setAirToggleState(airToggle, requestedAiring);
		airToggle.disabled = true;
		try {
			const response = await fetch(`/api/channels/${channel.id}/airing`, {
				method: "PUT",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({ airing: requestedAiring }),
			});
			const saved = await response.json();
			if (!response.ok) {
				throw new Error(saved.error || "Could not update channel airing state");
			}
			channels = channels.map((item) => (item.id === saved.id ? saved : item));
			setAirToggleState(airToggle, saved.airing);
			scanStatus.textContent = saved.airing
				? `${saved.name} is now airing; its schedule is regenerating.`
				: `${saved.name} is off air and removed from channel surfing.`;
		} catch (error) {
			setAirToggleState(airToggle, wasAiring);
			airToggle.classList.add("air-toggle-error");
			airToggle.querySelector(".air-toggle-label").textContent = "Save failed";
			airToggle.title = error.message;
			scanStatus.textContent = error.message;
			window.setTimeout(() => {
				airToggle.classList.remove("air-toggle-error");
				setAirToggleState(airToggle, wasAiring);
			}, 2200);
		} finally {
			airToggle.disabled = false;
		}
	});
  handle.addEventListener("dragstart", (event) => {
    draggedChannelID = channel.id;
    row.classList.add("dragging");
    event.dataTransfer.effectAllowed = "move";
    event.dataTransfer.setData("text/plain", String(channel.id));
  });
  handle.addEventListener("dragend", () => {
    draggedChannelID = null;
    row.classList.remove("dragging");
    channelRows.querySelectorAll(".drop-target").forEach((target) => target.classList.remove("drop-target"));
  });
  row.addEventListener("dragover", (event) => {
    if (!draggedChannelID || draggedChannelID === channel.id) {
      return;
    }
    event.preventDefault();
    event.dataTransfer.dropEffect = "move";
    row.classList.add("drop-target");
  });
  row.addEventListener("dragleave", () => row.classList.remove("drop-target"));
  row.addEventListener("drop", async (event) => {
    event.preventDefault();
    row.classList.remove("drop-target");
    const sourceID = Number(event.dataTransfer.getData("text/plain") || draggedChannelID);
    if (!sourceID || sourceID === channel.id) {
      return;
    }
    await moveChannelBefore(sourceID, channel.id);
  });
  row.querySelector(".edit-channel").addEventListener("click", () => editChannel(channel));
  row.querySelector(".delete-channel").addEventListener("click", (event) => openDeleteChannelModal(channel, event.currentTarget));
  return row;
}

function setAirToggleState(button, airing) {
  button.setAttribute("aria-checked", String(Boolean(airing)));
  button.querySelector(".air-toggle-label").textContent = airing ? "On Air" : "Off Air";
}

async function moveChannelBefore(sourceID, targetID) {
  const sourceIndex = channels.findIndex((channel) => channel.id === sourceID);
  const targetIndex = channels.findIndex((channel) => channel.id === targetID);
  if (sourceIndex < 0 || targetIndex < 0 || sourceIndex === targetIndex) {
    return;
  }

  const previousChannels = channels;
  const reordered = [...channels];
  const [source] = reordered.splice(sourceIndex, 1);
  const adjustedTargetIndex = reordered.findIndex((channel) => channel.id === targetID);
  reordered.splice(adjustedTargetIndex, 0, source);
  channels = reordered;
  renderChannels();

  try {
    const response = await fetch("/api/channels/reorder", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ channel_ids: channels.map((channel) => channel.id) }),
    });
    const payload = await response.json();
    if (!response.ok) {
      throw new Error(payload.error || "Could not reorder channels");
    }
    channels = payload.channels || channels;
    renderChannels();
  } catch (error) {
    channels = previousChannels;
    renderChannels();
    scanStatus.textContent = error.message;
  }
}

function upsertChannel(channel) {
  const index = channels.findIndex((existing) => existing.id === channel.id);
  if (index === -1) {
    channels = [...channels, channel];
    return;
  }

  channels = channels.map((existing) => (existing.id === channel.id ? channel : existing));
}

function clearBuilderLogoPreviewURL() {
  if (builderLogoPreviewURL) {
    URL.revokeObjectURL(builderLogoPreviewURL);
    builderLogoPreviewURL = null;
  }
}

function logoURLWithVersion(path, version = builderLogoPreviewVersion) {
  if (!path || !version) {
    return path || "";
  }
  const separator = path.includes("?") ? "&" : "?";
  return `${path}${separator}v=${version}`;
}

function newChannel() {
  clearBuilderLogoPreviewURL();
  builderLogoPreviewVersion = 0;
  currentChannel = null;
  channelNameInput.value = "";
  channelDescriptionInput.value = "";
  channelLogoPathInput.value = "";
  updateBuilderLogo();
  setChannelSaveStatus("");
  setView("channel-builder");
}

function editChannel(channel) {
  clearBuilderLogoPreviewURL();
  builderLogoPreviewVersion = 0;
  currentChannel = channel;
  channelNameInput.value = channel.name || "";
  channelDescriptionInput.value = channel.description || "";
  channelLogoPathInput.value = channel.logo_path || "";
  updateBuilderLogo();
  setChannelSaveStatus("");
  setView("channel-builder");
}

function ruleRowFromRule(rule) {
  const row = createRuleRow();
  const fieldLabel = ruleLabelForField(rule.field);
  const [mediaTypeSelect, fieldSelect, operatorSelect, valueSelect] = row.querySelectorAll("select");
  const mediaType = mediaTypeForRule(rule, currentChannel?.content_type);
  populateSelect(mediaTypeSelect, ruleMediaTypes, mediaType);
  populateSelect(fieldSelect, valuesWithExisting(availableRuleLabels(mediaType), fieldLabel), fieldLabel);
  updateRuleRow(row);
  populateSelect(operatorSelect, ruleOptions[fieldLabel].operators, rule.operator);
  populateSelect(valueSelect, valuesWithExisting(ruleValuesFor(mediaType, fieldLabel), rule.value), rule.value);
  return row;
}

function resetProgrammingState(channel = null) {
  programmingBlocks = normalizeProgrammingItems(channel?.programming_blocks, "block");
  programmingEvents = normalizeProgrammingItems(channel?.programming_events, "event");
  if (programmingBlocks.length === 0 && programmingEvents.length === 0) {
    programmingBlocks = [
      createProgrammingBlock({
        name: "Morning Cartoons",
        start_time: "06:00",
        end_time: "09:00",
        days: ["mon", "tue", "wed", "thu", "fri", "sat", "sun"],
        rule_set: "Use Content Rules",
      }),
    ];
    programmingEvents = [
      createProgrammingEvent({
        name: "Featured Program",
        time: "20:00",
        days: ["sun"],
        program: "Select a show or movie",
      }),
    ];
  }
  selectedProgrammingType = programmingBlocks.length > 0 ? "block" : "event";
  selectedProgrammingID =
    selectedProgrammingType === "block" ? programmingBlocks[0]?.id : programmingEvents[0]?.id;
}

function normalizeProgrammingItems(items, type) {
  if (!Array.isArray(items)) {
    return [];
  }
  return items.map((item) => (type === "event" ? createProgrammingEvent(item) : createProgrammingBlock(item)));
}

function createProgrammingBlock(overrides = {}) {
	const placement = overrides.placement || "timeline";
  return {
    id: overrides.id || `block-${Date.now()}-${Math.random().toString(16).slice(2)}`,
    type: "block",
    name: overrides.name || overrides.title || "New Programming Block",
    start_time: placement === "all_day" ? "" : (overrides.start_time || "06:00"),
    end_time: placement === "all_day" ? "" : (overrides.end_time || "09:00"),
    placement,
    days: Array.isArray(overrides.days) && overrides.days.length > 0 ? overrides.days : ["mon", "tue", "wed", "thu", "fri"],
    content_source: overrides.content_source || "rules",
    collection_id: Number(overrides.collection_id || 0),
    rule_set: overrides.rule_set || "Use Content Rules",
    schedule_mode: overrides.schedule_mode || "weekly",
    exact_date: overrides.exact_date || "",
    excluded_dates: Array.isArray(overrides.excluded_dates) ? overrides.excluded_dates : [],
    playback_mode: overrides.playback_mode || "shuffle",
    media_items: Array.isArray(overrides.media_items) ? overrides.media_items : [],
    priority: Number(overrides.priority || 0),
    enabled: overrides.active !== false && overrides.enabled !== false,
    active: overrides.active !== false && overrides.enabled !== false,
  };
}

function createProgrammingEvent(overrides = {}) {
  return {
    id: overrides.id || `event-${Date.now()}-${Math.random().toString(16).slice(2)}`,
    type: "event",
    name: overrides.name || "New Featured Program",
    time: overrides.time || "20:00",
    days: Array.isArray(overrides.days) && overrides.days.length > 0 ? overrides.days : ["fri"],
    program: overrides.program || "Select a show or movie",
    playback: overrides.playback || "continue",
    priority: Number(overrides.priority || 10),
    active: overrides.active !== false,
  };
}

function renderBuilderTabs() {
  builderTabButtons.forEach((button) => {
    const active = button.dataset.builderTab === activeBuilderTab;
    button.classList.toggle("active", active);
    button.setAttribute("aria-selected", active ? "true" : "false");
  });
  builderTabPanels.forEach((panel) => {
    const active = panel.dataset.builderPanel === activeBuilderTab;
    panel.hidden = !active;
    panel.classList.toggle("active", active);
  });
}

function setBuilderTab(tab) {
  activeBuilderTab = tab === "programming" ? "programming" : "content";
  renderBuilderTabs();
  if (activeBuilderTab === "programming") {
    renderProgrammingBuilder();
  }
}

function renderProgrammingBuilder() {
  renderProgrammingList(programmingBlockList, programmingBlocks, "block");
  renderProgrammingList(programmingEventList, programmingEvents, "event");
  renderProgrammingEditor();
}

function renderProgrammingList(container, items, type) {
  if (!container) {
    return;
  }
  if (items.length === 0) {
    const empty = document.createElement("p");
    empty.className = "programming-empty";
    empty.textContent = type === "block" ? "No programming blocks yet." : "No featured events yet.";
    container.replaceChildren(empty);
    return;
  }
  container.replaceChildren(...items.map((item) => programmingListRow(item, type)));
}

function programmingListRow(item, type) {
  const row = document.createElement("article");
  row.className = "programming-row";
  row.classList.toggle("active", selectedProgrammingType === type && selectedProgrammingID === item.id);
  row.innerHTML = `
    <span class="programming-row-icon">${type === "block" ? ruleIcons.tv : ruleIcons.rating}</span>
    <button class="programming-row-body" type="button">
      <strong>${escapeHTML(item.name)}</strong>
      <span>${escapeHTML(programmingScheduleSummary(item, type))}</span>
      <small>Fills with: ${escapeHTML(type === "block" ? programmingBlockMediaSummary(item) : item.program)}</small>
    </button>
    <span class="programming-status">${item.active ? "Active" : "Paused"}</span>
    <button class="icon-button programming-menu-button" type="button" aria-label="Edit ${escapeAttribute(item.name)}">
      <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 5h.01M12 12h.01M12 19h.01" /></svg>
    </button>
  `;
  row.querySelector(".programming-row-body").addEventListener("click", () => selectProgrammingItem(type, item.id));
  row.querySelector(".programming-menu-button").addEventListener("click", () => selectProgrammingItem(type, item.id));
  return row;
}

function programmingScheduleSummary(item, type) {
  const days = formatProgrammingDays(item.days);
  if (type === "event") {
    return `${days} • ${formatTimeLabel(item.time)} • ${item.playback === "restart" ? "Restart Program" : "Continue Series"}`;
  }
  return `${days} • ${formatTimeLabel(item.start_time)} - ${formatTimeLabel(item.end_time)}`;
}

function programmingBlockMediaSummary(item) {
  if (item.media_items?.length > 0) {
    return `${item.media_items.length} selected media item${item.media_items.length === 1 ? "" : "s"}`;
  }
  return item.rule_set;
}

function formatProgrammingDays(days = []) {
  const allDays = ["mon", "tue", "wed", "thu", "fri", "sat", "sun"];
  if (allDays.every((day) => days.includes(day))) {
    return "Every Day";
  }
  if (["mon", "tue", "wed", "thu", "fri"].every((day) => days.includes(day)) && !days.includes("sat") && !days.includes("sun")) {
    return "Mon - Fri";
  }
  return days.map((day) => dayLabel(day)).join(", ") || "No days";
}

function dayLabel(day) {
  return {
    mon: "Mon",
    tue: "Tue",
    wed: "Wed",
    thu: "Thu",
    fri: "Fri",
    sat: "Sat",
    sun: "Sun",
  }[day] || day;
}

function formatTimeLabel(value) {
  if (!value) {
    return "";
  }
  const [hourText, minute = "00"] = value.split(":");
  let hour = Number(hourText);
  if (!Number.isFinite(hour)) {
    return value;
  }
  const suffix = hour >= 12 ? "PM" : "AM";
  hour = hour % 12 || 12;
  return `${hour}:${minute} ${suffix}`;
}

function selectProgrammingItem(type, id) {
  selectedProgrammingType = type;
  selectedProgrammingID = id;
  renderProgrammingBuilder();
}

function selectedProgrammingItem() {
  const items = selectedProgrammingType === "event" ? programmingEvents : programmingBlocks;
  return items.find((item) => item.id === selectedProgrammingID) || items[0] || null;
}

function renderProgrammingEditor() {
  if (!programmingEditor) {
    return;
  }
  const item = selectedProgrammingItem();
  if (!item) {
    programmingEditor.innerHTML = `<div class="programming-editor-empty">Select a block or event to edit.</div>`;
    return;
  }
  if (selectedProgrammingType === "event") {
    renderProgrammingEventEditor(item);
    return;
  }
  renderProgrammingBlockEditor(item);
}

function renderProgrammingBlockEditor(item) {
  programmingEditor.innerHTML = `
    <div class="programming-editor-head">
      <span class="programming-editor-icon">${ruleIcons.calendar || ruleIcons.tv}</span>
      <div>
        <h3>${escapeHTML(item.name)}</h3>
        <p>Programming Block</p>
      </div>
    </div>
    <div class="programming-editor-tabs"><button class="active" type="button">Details</button><button type="button">Playback</button></div>
    <label><span>Name</span><input data-programming-field="name" type="text" value="${escapeAttribute(item.name)}" /></label>
    <div class="programming-time-grid">
      <label><span>Start</span><input data-programming-field="start_time" type="time" value="${escapeAttribute(item.start_time)}" /></label>
      <label><span>End</span><input data-programming-field="end_time" type="time" value="${escapeAttribute(item.end_time)}" /></label>
    </div>
    ${programmingDaysHTML(item.days)}
    <label><span>Content Source</span><select data-programming-field="content_source"><option value="rules">Use Content Rules</option><option value="specific">Specific Rule Set</option></select></label>
    <label><span>Rule Set</span><input data-programming-field="rule_set" type="text" value="${escapeAttribute(item.rule_set)}" /></label>
    <div class="programming-editor-media">
      <span>Media</span>
      <p>${escapeHTML(programmingBlockMediaSummary(item))}</p>
    </div>
    <label><span>Priority</span><input data-programming-field="priority" type="number" min="0" value="${escapeAttribute(item.priority)}" /></label>
    <p class="programming-editor-help">Higher numbers override lower priority blocks.</p>
    <button class="button danger ghost programming-delete-button" type="button">Delete Block</button>
  `;
  programmingEditor.querySelector('[data-programming-field="content_source"]').value = item.content_source;
  bindProgrammingEditor(item);
}

function renderProgrammingEventEditor(item) {
  programmingEditor.innerHTML = `
    <div class="programming-editor-head">
      <span class="programming-editor-icon event">${ruleIcons.rating}</span>
      <div>
        <h3>${escapeHTML(item.name)}</h3>
        <p>Featured Program Event</p>
      </div>
    </div>
    <div class="programming-editor-tabs"><button class="active" type="button">Details</button><button type="button">Playback</button></div>
    <label><span>Name</span><input data-programming-field="name" type="text" value="${escapeAttribute(item.name)}" /></label>
    <label><span>Program</span><input data-programming-field="program" type="text" value="${escapeAttribute(item.program)}" /></label>
    <label><span>Time</span><input data-programming-field="time" type="time" value="${escapeAttribute(item.time)}" /></label>
    ${programmingDaysHTML(item.days)}
    <label><span>Playback</span><select data-programming-field="playback"><option value="continue">Continue Series</option><option value="restart">Restart Program</option></select></label>
    <label><span>Priority</span><input data-programming-field="priority" type="number" min="0" value="${escapeAttribute(item.priority)}" /></label>
    <p class="programming-editor-help">Events override programming blocks at the same time.</p>
    <button class="button danger ghost programming-delete-button" type="button">Delete Event</button>
  `;
  programmingEditor.querySelector('[data-programming-field="playback"]').value = item.playback;
  bindProgrammingEditor(item);
}

function programmingDaysHTML(days = []) {
  return `
    <div class="programming-days" aria-label="Days">
      ${["mon", "tue", "wed", "thu", "fri", "sat", "sun"]
        .map(
          (day) => `
            <button class="${days.includes(day) ? "active" : ""}" type="button" data-programming-day="${day}">
              ${dayLabel(day)}
            </button>
          `,
        )
        .join("")}
    </div>
  `;
}

function bindProgrammingEditor(item) {
  const updateField = (input) => {
    const field = input.dataset.programmingField;
    item[field] = field === "priority" ? Number(input.value || 0) : input.value;
    renderProgrammingList(programmingBlockList, programmingBlocks, "block");
    renderProgrammingList(programmingEventList, programmingEvents, "event");
  };
  programmingEditor.querySelectorAll("[data-programming-field]").forEach((input) => {
    input.addEventListener("input", () => updateField(input));
    input.addEventListener("change", () => updateField(input));
  });
  programmingEditor.querySelectorAll("[data-programming-day]").forEach((button) => {
    button.addEventListener("click", () => {
      const day = button.dataset.programmingDay;
      item.days = item.days.includes(day) ? item.days.filter((existing) => existing !== day) : [...item.days, day];
      renderProgrammingBuilder();
    });
  });
  programmingEditor.querySelector(".programming-delete-button")?.addEventListener("click", () => {
    if (selectedProgrammingType === "event") {
      programmingEvents = programmingEvents.filter((event) => event.id !== item.id);
      selectedProgrammingID = programmingEvents[0]?.id || programmingBlocks[0]?.id || null;
      selectedProgrammingType = programmingEvents[0] ? "event" : "block";
    } else {
      programmingBlocks = programmingBlocks.filter((block) => block.id !== item.id);
      selectedProgrammingID = programmingBlocks[0]?.id || programmingEvents[0]?.id || null;
      selectedProgrammingType = programmingBlocks[0] ? "block" : "event";
    }
    renderProgrammingBuilder();
  });
}

function addProgrammingBlock() {
  openProgrammingBlockModal();
}

function openProgrammingBlockModal() {
  if (!programmingBlockModal) {
    return;
  }
  pendingProgrammingBlockDays = ["mon", "tue", "wed", "thu", "fri"];
  pendingProgrammingBlockMedia = [];
  if (programmingBlockTitleInput) {
    programmingBlockTitleInput.value = "";
  }
  if (programmingBlockStartInput) {
    programmingBlockStartInput.value = "06:00";
  }
  if (programmingBlockStopInput) {
    programmingBlockStopInput.value = "09:00";
  }
  renderProgrammingBlockModalDays();
  renderProgrammingBlockMediaPicker();
  programmingBlockModal.hidden = false;
  programmingBlockModal.classList.add("open");
  programmingBlockModal.setAttribute("aria-hidden", "false");
  programmingBlockTitleInput?.focus();
}

function closeProgrammingBlockModal() {
  if (!programmingBlockModal) {
    return;
  }
  programmingBlockModal.classList.remove("open");
  programmingBlockModal.setAttribute("aria-hidden", "true");
  programmingBlockModal.hidden = true;
}

function renderProgrammingBlockModalDays() {
  programmingBlockDays?.querySelectorAll("[data-block-day]").forEach((button) => {
    button.classList.toggle("active", pendingProgrammingBlockDays.includes(button.dataset.blockDay));
  });
}

function setProgrammingBlockWeekdays() {
  pendingProgrammingBlockDays = ["mon", "tue", "wed", "thu", "fri"];
  renderProgrammingBlockModalDays();
}

function togglePendingProgrammingBlockDay(day) {
  pendingProgrammingBlockDays = pendingProgrammingBlockDays.includes(day)
    ? pendingProgrammingBlockDays.filter((existing) => existing !== day)
    : [...pendingProgrammingBlockDays, day];
  renderProgrammingBlockModalDays();
}

function mediaPickerItems() {
  const movieItems = movies.map((movie) => ({
    id: `movie:${movie.id}`,
    media_type: "movie",
    media_id: movie.id,
    title: movie.title,
    detail: movie.year ? `Movie • ${movie.year}` : "Movie",
  }));
  const tvItems = tvShows.map((show) => ({
    id: `tv:${show.id}`,
    media_type: "tv",
    media_id: show.id,
    title: show.title,
    detail: show.year ? `TV Show • ${show.year}` : "TV Show",
  }));
  return [...tvItems, ...movieItems].filter((item) => item.media_id);
}

function renderProgrammingBlockMediaPicker() {
  if (!programmingBlockMediaList) {
    return;
  }
  const items = mediaPickerItems();
  if (items.length === 0) {
    programmingBlockMediaList.innerHTML = `<p class="programming-empty">No local movies or TV shows loaded yet.</p>`;
    updateProgrammingBlockMediaCount();
    return;
  }
  programmingBlockMediaList.replaceChildren(
    ...items.map((item) => {
      const row = document.createElement("label");
      row.className = "programming-media-option";
      row.innerHTML = `
        <input type="checkbox" value="${escapeAttribute(item.id)}" />
        <span>
          <strong>${escapeHTML(item.title)}</strong>
          <small>${escapeHTML(item.detail)}</small>
        </span>
      `;
      const checkbox = row.querySelector("input");
      checkbox.checked = pendingProgrammingBlockMedia.some((media) => media.id === item.id);
      checkbox.addEventListener("change", () => toggleProgrammingBlockMedia(item, checkbox.checked));
      return row;
    }),
  );
  updateProgrammingBlockMediaCount();
}

function toggleProgrammingBlockMedia(item, selected) {
  if (selected) {
    if (!pendingProgrammingBlockMedia.some((media) => media.id === item.id)) {
      pendingProgrammingBlockMedia = [...pendingProgrammingBlockMedia, item];
    }
  } else {
    pendingProgrammingBlockMedia = pendingProgrammingBlockMedia.filter((media) => media.id !== item.id);
  }
  updateProgrammingBlockMediaCount();
}

function updateProgrammingBlockMediaCount() {
  if (!programmingBlockMediaCount) {
    return;
  }
  const count = pendingProgrammingBlockMedia.length;
  programmingBlockMediaCount.textContent = `${count} selected`;
}

function createProgrammingBlockFromModal() {
  const title = programmingBlockTitleInput?.value.trim() || "";
  if (!title) {
    programmingBlockTitleInput?.focus();
    return;
  }
  const block = createProgrammingBlock({
    name: title,
    days: pendingProgrammingBlockDays,
    start_time: programmingBlockStartInput?.value || "06:00",
    end_time: programmingBlockStopInput?.value || "09:00",
    content_source: pendingProgrammingBlockMedia.length > 0 ? "specific" : "rules",
    rule_set: pendingProgrammingBlockMedia.length > 0 ? "Selected Media" : "Use Content Rules",
    media_items: pendingProgrammingBlockMedia,
  });
  programmingBlocks = [...programmingBlocks, block];
  closeProgrammingBlockModal();
  selectProgrammingItem("block", block.id);
}

function addProgrammingEvent() {
  const event = createProgrammingEvent();
  programmingEvents = [...programmingEvents, event];
  selectProgrammingItem("event", event.id);
}

async function loadChannels({ clearOnError = true } = {}) {
  try {
    const response = await fetch("/api/channels");
    const payload = await response.json();
    if (!response.ok) {
      throw new Error(payload.error || "Could not load channels");
    }
    channels = payload.channels || [];
    renderChannels();
    renderProgrammingPage();
  } catch (error) {
    if (clearOnError) {
      channels = [];
      renderChannels();
    }
    scanStatus.textContent = error.message;
  } finally {
    renderDashboard();
    if (channels.length > 0) loadDashboardNowPlaying();
  }
}

function renderChannelImportStatus(status) {
  if (!channelImportProgress || !channelImportProgressText) {
    return;
  }
  if (!status?.active) {
    channelImportProgress.hidden = true;
    if (status?.finished_at && status.total_items) {
      channelCount.textContent = status.errors
        ? `Generated schedules for ${status.total_items} channels with ${status.errors} errors`
        : `Generated schedules for ${status.total_items} channels`;
    }
    return;
  }

  const current = status.current || "Preparing schedules";
  const total = status.total_items || 0;
  const index = status.current_item || 0;
  const count = total ? `${index}/${total}` : "";
  channelImportProgressText.textContent = count
    ? `Generating ${current} (${count})`
    : `Generating ${current}`;
  channelImportProgress.hidden = false;
}

function stopChannelImportStatusPolling() {
  if (channelImportStatusTimer) {
    clearTimeout(channelImportStatusTimer);
    channelImportStatusTimer = null;
  }
}

async function pollChannelImportStatus() {
  stopChannelImportStatusPolling();
  try {
    const response = await fetch("/api/channels/import/status");
    const status = await readJSONResponse(response);
    if (!response.ok) {
      throw new Error(status.error || "Could not load channel import status");
    }
    renderChannelImportStatus(status);
    if (status.active) {
      channelImportStatusTimer = setTimeout(pollChannelImportStatus, 900);
    }
  } catch {
    channelImportStatusTimer = setTimeout(pollChannelImportStatus, 2000);
  }
}

async function importChannels() {
  const file = importChannelsFileInput?.files?.[0];
  if (!file) {
    return;
  }

  const form = new FormData();
  form.append("channels", file);
  importChannelsButton.disabled = true;
  channelCount.textContent = "Importing channels...";
  try {
    const response = await fetch("/api/channels/import", {
      method: "POST",
      body: form,
    });
    const payload = await readJSONResponse(response);
    if (!response.ok) {
      throw new Error(payload.error || "Could not import channels");
    }
    channels = payload.channels || [];
    renderChannels();
    channelCount.textContent = `Imported ${payload.imported || 0} channels (${payload.created || 0} new, ${payload.updated || 0} updated)`;
    pollChannelImportStatus();
  } catch (error) {
    channelCount.textContent = error.message;
  } finally {
    importChannelsButton.disabled = false;
    importChannelsFileInput.value = "";
  }
}

async function readJSONResponse(response) {
  const text = await response.text();
  if (!text) {
    return {};
  }
  try {
    return JSON.parse(text);
  } catch {
    return { error: text };
  }
}

async function saveChannel() {
  const name = channelNameInput.value.trim();
  if (!name) {
    setChannelSaveStatus("Channel name is required.");
    channelNameInput.focus();
    return;
  }

  const payload = {
    name,
    description: channelDescriptionInput.value.trim(),
    logo_path: channelLogoPathInput.value.trim(),
    content_type: currentChannel?.content_type || "mixed",
  };
  const url = currentChannel ? `/api/channels/${currentChannel.id}` : "/api/channels";
  const method = currentChannel ? "PUT" : "POST";

  saveChannelButton.disabled = true;
  setChannelSaveStatus("Saving channel...");
  try {
    const response = await fetch(url, {
      method,
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
    const saved = await response.json();
    if (!response.ok) {
      throw new Error(saved.error || "Could not save channel");
    }

    currentChannel = saved;
    upsertChannel(saved);
    renderChannels();
    setChannelSaveStatus("Channel saved.");
    setView("channels");
    await loadChannels({ clearOnError: false });
  } catch (error) {
    setChannelSaveStatus(error.message);
  } finally {
    saveChannelButton.disabled = false;
  }
}

function openDeleteChannelModal(channel, trigger) {
  if (!deleteChannelModal || !channel?.id) {
    return;
  }
  pendingDeleteChannel = { channel, trigger };
  deleteChannelMessage.textContent = `Delete “${channel.name}”? Its schedule and channel configuration will be permanently removed.`;
  deleteChannelModal.hidden = false;
  deleteChannelModal.classList.add("open");
  deleteChannelModal.setAttribute("aria-hidden", "false");
  window.requestAnimationFrame(() => confirmDeleteChannel?.focus());
}

function closeDeleteChannelModal({ restoreFocus = true } = {}) {
  const trigger = pendingDeleteChannel?.trigger;
  pendingDeleteChannel = null;
  deleteChannelModal?.classList.remove("open");
  deleteChannelModal?.setAttribute("aria-hidden", "true");
  if (deleteChannelModal) {
    deleteChannelModal.hidden = true;
  }
  if (restoreFocus) {
    trigger?.focus();
  }
}

async function deletePendingChannel() {
  const pending = pendingDeleteChannel;
  if (!pending) {
    return;
  }

  confirmDeleteChannel.disabled = true;
  confirmDeleteChannel.textContent = "Deleting...";
  try {
    const response = await fetch(`/api/channels/${pending.channel.id}`, { method: "DELETE" });
    if (!response.ok) {
      const payload = await response.json();
      throw new Error(payload.error || "Could not delete channel");
    }
    closeDeleteChannelModal({ restoreFocus: false });
    scanStatus.textContent = `${pending.channel.name} was deleted.`;
    await loadChannels();
  } catch (error) {
    scanStatus.textContent = error.message;
  } finally {
    confirmDeleteChannel.disabled = false;
    confirmDeleteChannel.textContent = "Delete Channel";
  }
}

function collectRules() {
  return [...document.querySelectorAll(".metadata-rule-row")]
    .map((row) => {
      const [mediaTypeSelect, fieldSelect, operatorSelect, valueSelect] = row.querySelectorAll("select");
      const options = ruleOptions[fieldSelect.value];
      return {
        media_type: mediaTypeSelect.value,
        field: options.field,
        operator: operatorSelect.value,
        value: valueSelect.value,
      };
    })
    .filter((rule) => rule.value);
}

function collectChannelContentType() {
  const mediaTypes = new Set(collectRules().map((rule) => rule.media_type).filter(Boolean));
  if (mediaTypes.size > 1) {
    return "mixed";
  }
  return mediaTypes.values().next().value || currentChannel?.content_type || "movie";
}

function collectProgrammingBlocks() {
  return programmingBlocks.map((block) => ({
    id: Number(block.id) || 0,
    title: block.name || block.title || "",
    name: block.name || block.title || "",
    start_time: block.start_time,
    end_time: block.end_time,
    days: block.days || [],
    content_source: block.media_items?.length > 0 ? "media" : block.content_source || "rules",
    rule_set: block.rule_set || "",
    priority: Number(block.priority || 0),
    enabled: block.active !== false,
    media_items: (block.media_items || []).map((item, index) => ({
      media_type: item.media_type,
      media_id: Number(item.media_id || 0),
      sort_order: index + 1,
      title: item.title || "",
    })),
  }));
}

function ruleLabelForField(field) {
  return Object.keys(ruleOptions).find((label) => ruleOptions[label].field === field) || "Genre";
}

function mediaTypeForRule(rule, fallback = "movie") {
  if (rule?.media_type === "movie" || rule?.media_type === "tv") {
    return rule.media_type;
  }
  if (fallback === "tv" || fallback === "movie") {
    return fallback;
  }
  if (rule?.field === "network") {
    return "tv";
  }
  return "movie";
}

function valuesWithExisting(values, existing) {
  if (!existing || values.some((value) => (typeof value === "string" ? value : value.value) === existing)) {
    return values;
  }
  return [...values, existing];
}

function availableRuleLabels(mediaType = "movie") {
  return ruleFieldsByMediaType[mediaType] || ruleFieldsByMediaType.movie;
}

function ruleValuesFor(mediaType, fieldLabel) {
  if (ruleOptions[fieldLabel]?.fixedValues) {
    return ruleOptions[fieldLabel].values;
  }
  const localValues = localRuleValuesFor(mediaType, fieldLabel);
  if (localValues) {
    return localValues;
  }
  return ruleOptions[fieldLabel]?.values || [];
}

function localRuleValuesFor(mediaType, fieldLabel) {
  const loaded = mediaType === "tv" ? tvShowsLoaded : moviesLoaded;
  if (!loaded) {
    return null;
  }
  const records = mediaType === "tv" ? tvShows : movies;
  return uniqueSorted(
    records.flatMap((record) => valuesForRuleField(record, fieldLabel, mediaType)),
  );
}

function valuesForRuleField(record, fieldLabel, mediaType) {
  switch (fieldLabel) {
    case "Title":
      return [record.title];
    case "Genre":
      return record.genres || [];
    case "Year":
      return [record.year];
    case "Runtime":
      if (mediaType === "tv") {
        return (record.episodeMetadata || []).map((episode) => episode.runtime_minutes || episode.runtime);
      }
      return [record.runtimeMinutes];
    case "Rating":
      return [record.ratingValue ?? record.rating];
    case "Network":
      return [record.network];
    case "Studio":
      return [record.studio];
    case "Actor":
      return record.actors || [];
    case "Director":
      return record.directors || [];
    case "Writer":
      return record.writers || [];
    case "Country":
      return record.countries || [];
    case "Language":
      return record.languages || [];
    case "Series Status":
      return [record.seriesStatus];
    case "First Air Date":
      return [record.firstAirDate];
    case "Release Date":
      return [record.releaseDate];
    case "Date Added":
      return [dateOnly(record.added)];
    case "Tags":
      return record.tags || [];
    default:
      return [];
  }
}

function uniqueSorted(values) {
  return [...new Set(values.map((value) => String(value || "").trim()).filter(Boolean))].sort((a, b) =>
    a.localeCompare(b, undefined, { numeric: true, sensitivity: "base" }),
  );
}

function dateOnly(value) {
  if (!value) {
    return "";
  }
  return String(value).slice(0, 10);
}

function channelLogoText(name) {
  return (name || "TV").trim().slice(0, 2).toUpperCase();
}

function updateBuilderLogo() {
  const logoSource = builderLogoPreviewURL || logoURLWithVersion(channelLogoPathInput.value);
  builderLogoPreview.textContent = logoSource ? "" : channelLogoText(channelNameInput.value);
  builderLogoPreview.classList.toggle("logo-image", Boolean(logoSource));
  if (logoSource) {
    builderLogoPreview.style.setProperty("--logo", `url("${escapeAttribute(logoSource)}")`);
  } else {
    builderLogoPreview.style.removeProperty("--logo");
  }
}

async function uploadChannelLogo() {
  const file = channelLogoFileInput?.files?.[0];
  if (!file) {
    return;
  }

  clearBuilderLogoPreviewURL();
  builderLogoPreviewURL = URL.createObjectURL(file);
  updateBuilderLogo();

  const form = new FormData();
  form.append("logo", file);
  if (currentChannel?.id) {
    form.append("channel_id", String(currentChannel.id));
  }
  channelLogoEditButton.disabled = true;
  setChannelSaveStatus("Uploading logo...");
  try {
    const response = await fetch("/api/channel-logos", {
      method: "POST",
      body: form,
    });
    const payload = await response.json();
    if (!response.ok) {
      throw new Error(payload.error || "Could not upload logo");
    }

    channelLogoPathInput.value = payload.logo_path || "";
    builderLogoPreviewVersion = Date.now();
    clearBuilderLogoPreviewURL();
    updateBuilderLogo();
    setChannelSaveStatus("Logo uploaded. Save the channel to use it.");
  } catch (error) {
    clearBuilderLogoPreviewURL();
    updateBuilderLogo();
    setChannelSaveStatus(error.message);
  } finally {
    channelLogoEditButton.disabled = false;
    channelLogoFileInput.value = "";
  }
}

function setChannelSaveStatus(message) {
  if (channelSaveStatus) {
    channelSaveStatus.textContent = message;
  }
}

function channelLogoMarkup(name, logoPath) {
  if (logoPath) {
    return `<div class="channel-logo logo-image" style="--logo: url('${escapeAttribute(logoPath)}')"></div>`;
  }
  return `<div class="channel-logo">${escapeHTML(channelLogoText(name))}</div>`;
}

function setupRuleBuilder() {
  document.querySelectorAll(".metadata-rule-row").forEach((row) => {
    setupRuleRow(row);
  });
  updateRuleCount();
}

function setupRuleRow(row) {
  const [mediaTypeSelect, fieldSelect] = row.querySelectorAll("select");
  if (!fieldSelect) {
    return;
  }

  populateSelect(mediaTypeSelect, ruleMediaTypes, mediaTypeSelect.value || "movie");
  populateSelect(fieldSelect, availableRuleLabels(mediaTypeSelect.value), fieldSelect.value);
  updateRuleRow(row);

  if (!mediaTypeSelect.dataset.ruleListener) {
    mediaTypeSelect.addEventListener("change", () => updateRuleRow(row));
    mediaTypeSelect.dataset.ruleListener = "true";
  }

  if (!fieldSelect.dataset.ruleListener) {
    fieldSelect.addEventListener("change", () => updateRuleRow(row));
    fieldSelect.dataset.ruleListener = "true";
  }

  const deleteButton = row.querySelector(".delete-rule-button");
  if (!deleteButton) {
    return;
  }
  if (!deleteButton.dataset.ruleListener) {
    deleteButton.addEventListener("click", () => openDeleteRuleModal(row));
    deleteButton.dataset.ruleListener = "true";
  }
}

function updateRuleRow(row) {
  const [mediaTypeSelect, fieldSelect, operatorSelect, valueSelect] = row.querySelectorAll("select");
  const mediaType = mediaTypeSelect.value || "movie";
  populateSelect(fieldSelect, availableRuleLabels(mediaType), fieldSelect.value);
  const field = fieldSelect.value;
  const options = ruleOptions[field];

  if (!options) {
    return;
  }

  populateSelect(operatorSelect, options.operators, operatorSelect.value);
  populateSelect(valueSelect, ruleValuesFor(mediaType, field), valueSelect.value);
  row.dataset.mediaType = mediaType;
  row.dataset.field = field.toLowerCase();

  const mediaIcon = row.querySelector(".rule-media-icon");
  if (mediaIcon) {
    mediaIcon.innerHTML = ruleIcons[mediaType] || ruleIcons.movie;
  }
}

function populateSelect(select, values, preferredValue) {
  select.disabled = values.length === 0;
  if (values.length === 0) {
    const option = document.createElement("option");
    option.value = "";
    option.textContent = "No local values";
    select.replaceChildren(option);
    return;
  }

  const normalizedValues = values.map((value) =>
    typeof value === "string" ? { value, label: value } : value,
  );
  const optionValues = normalizedValues.map((option) => option.value);
  const value = optionValues.includes(preferredValue) ? preferredValue : normalizedValues[0].value;
  select.replaceChildren(
    ...normalizedValues.map((optionValue) => {
      const option = document.createElement("option");
      option.value = optionValue.value;
      option.textContent = optionValue.label;
      option.selected = optionValue.value === value;
      return option;
    }),
  );
}

function addRule() {
  if (!rulesList) {
    return;
  }

  const row = createRuleRow();
  rulesList.append(row);
  setupRuleRow(row);
  updateRuleCount();
}

function createRuleRow() {
  const row = document.createElement("article");
  row.className = "rule-row metadata-rule-row";
  row.innerHTML = `
    <span class="drag-handle">••</span>
    <label class="rule-select-wrap media-type-select">
      <span class="rule-media-icon"></span>
      <select aria-label="Rule media type"></select>
    </label>
    <select aria-label="Rule field"></select>
    <select aria-label="Rule operator"></select>
    <select aria-label="Rule value"></select>
    <button class="icon-button delete-rule-button" type="button" aria-label="Delete rule">
      <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M3 6h18M8 6V4h8v2M6 6l1 15h10l1-15" /></svg>
    </button>
  `;
  return row;
}

function updateRuleCount() {
  if (!ruleCountSummary) {
    return;
  }

  const count = document.querySelectorAll(".metadata-rule-row").length;
  ruleCountSummary.textContent = `${count} Active`;
}

function openDeleteRuleModal(row) {
  if (!deleteRuleModal) {
    row.remove();
    updateRuleCount();
    return;
  }

  pendingDeleteRule = row;
  deleteRuleModal.hidden = false;
  deleteRuleModal.classList.add("open");
  deleteRuleModal.setAttribute("aria-hidden", "false");
  confirmDeleteRule?.focus();
}

function closeDeleteRuleModal() {
  pendingDeleteRule = null;
  deleteRuleModal?.classList.remove("open");
  deleteRuleModal?.setAttribute("aria-hidden", "true");
  if (deleteRuleModal) {
    deleteRuleModal.hidden = true;
  }
}

function deletePendingRule() {
  if (pendingDeleteRule) {
    pendingDeleteRule.remove();
    updateRuleCount();
  }
  closeDeleteRuleModal();
}

async function loadMetadataFacets() {
  try {
    const response = await fetch("/api/metadata/facets");
    const facets = await response.json();
    if (!response.ok) {
      throw new Error(facets.error || "Could not load local metadata");
    }

    ruleOptions.Title.values = facets.titles || [];
    ruleOptions.Genre.values = facets.genres || [];
    ruleOptions.Year.values = (facets.release_years || []).map(String);
    ruleOptions.Runtime.values = (facets.runtimes || []).map(String);
    ruleOptions.Rating.values = facets.ratings || [];
    ruleOptions.Studio.values = facets.studios || [];
    ruleOptions.Network.values = facets.networks || [];
    ruleOptions.Actor.values = facets.actors || [];
    ruleOptions.Director.values = facets.directors || [];
    ruleOptions.Writer.values = facets.writers || [];
    ruleOptions.Country.values = facets.countries || [];
    ruleOptions.Language.values = facets.languages || [];
    ruleOptions["Series Status"].values = facets.series_statuses || [];
    ruleOptions["Release Date"].values = facets.release_dates || [];
    ruleOptions["First Air Date"].values = facets.first_air_dates || [];
    ruleOptions["Date Added"].values = facets.date_added_dates || [];
    ruleOptions.Tags.values = facets.tags || [];
    setupRuleBuilder();
  } catch (error) {
    scanStatus.textContent = error.message;
    setupRuleBuilder();
  }
}

function renderMovieCard(movie) {
  const card = document.createElement("article");
  card.className = `movie-card${movie === selectedMovie ? " selected" : ""}${movie.missing ? " missing" : ""}`;

  const poster = document.createElement("button");
  poster.className = "poster-button";
  poster.type = "button";
  poster.style.setProperty("--poster", movie.poster);
  poster.addEventListener("click", () => {
    selectedMovie = movie;
    render();
  });

  const status = document.createElement("span");
  if (movie.missing) {
    status.className = "tv-presence-badge missing";
    status.textContent = "Missing";
  } else if (movie.duplicate) {
    status.className = "tv-presence-badge duplicate";
    status.textContent = "Possible Duplicate";
  } else {
    status.className = "match-check";
    status.textContent = "✓";
  }
  poster.append(status);

  const title = document.createElement("p");
  title.className = "movie-title";
  title.textContent = movie.title;

  const year = document.createElement("p");
  year.className = "movie-year";
  year.innerHTML = `<span>${escapeHTML(movie.year || "Unknown")}</span><span class="content-rating">${escapeHTML(movie.rating || "NR")}</span>`;

  card.append(poster, title, year);
  return card;
}

function renderDetails(movie) {
  if (!movie) {
    detailsPanel.innerHTML = `
      <section class="details-empty">
        <h2>No movie selected</h2>
        <p>Scanned movies saved in the database will appear here.</p>
      </section>
    `;
    return;
  }

  const tags = movie.tags.map((tag) => `<span class="tag">${escapeHTML(tag)}</span>`).join("");
  const genres = movie.genres.join(", ");
  const tmdb = movie.tmdb || {};
  const technical = mediaTechnicalSummary(movie.mediaMetadata);

  detailsPanel.innerHTML = `
    ${movie.missing ? `
      <section class="tv-reconciliation-warning">
        <strong>Media file is missing</strong>
        <p>RetroTV last saw this movie ${escapeHTML(formatDate(movie.lastSeenAt))}. Its previous path was <code>${escapeHTML(movie.filePath)}</code>.</p>
        ${movie.renameCandidate ? `<p>A current record named <strong>${escapeHTML(movie.renameCandidate.title)}</strong> may be the renamed replacement.</p>` : ""}
        <p>Removing this entry deletes only its RetroTV database record and scheduled entries. It never deletes the movie file.</p>
      </section>` : movie.duplicate ? `
      <section class="tv-reconciliation-warning duplicate">
        <strong>Possible duplicate record</strong>
        <p>Another movie record has the same title, year, or TMDB identity. Review the file paths before removing either record.</p>
      </section>` : ""}
    <div class="details-head">
      <div class="details-poster"></div>
      <div class="details-title">
        <h2>${escapeHTML(movie.title)}</h2>
        <div class="meta-line">
          <span>${movie.year || "Unknown"}</span>
          <span>${escapeHTML(movie.rating || "NR")}</span>
          <span>${escapeHTML(movie.runtime || "Unknown")}</span>
        </div>
        <div class="meta-line">${escapeHTML(genres)}</div>
        <div class="rating-line">
          <span class="star">★</span>
          <strong>${tmdb.voteAverage || "0.0"}</strong>
          <span>/ 10</span>
          <span>TMDB</span>
        </div>
      </div>
      <div class="details-actions">
        <button class="button primary compact add-to-collection-button" type="button" title="Collection management is not yet available">Add to Collection</button>
        <button class="button danger compact delete-movie-button" type="button">${movie.missing ? "Remove Record" : "Delete"}</button>
        <button class="details-close" type="button" aria-label="Close details">×</button>
      </div>
    </div>

    <section class="details-section">
      <h3>Overview</h3>
      <p>${escapeHTML(movie.overview)}</p>
    </section>

    <section class="details-section">
      <h3>Details</h3>
      <div class="detail-list">
        ${detailRow("Original Title", movie.originalTitle || movie.title)}
        ${detailRow("Source", movie.source)}
        ${detailRow("Studio", movie.studio || "Unknown")}
        ${detailRow("File Path", movie.filePath)}
        ${detailRow("Date Added", formatDate(movie.added))}
        ${detailRow("Last Modified", formatDate(movie.modified))}
        ${detailRow("File Size", technical.fileSize)}
        ${detailRow("Video", technical.video)}
        ${detailRow("Audio", technical.audio)}
        ${detailRow("Subtitles", technical.subtitles)}
        ${technical.error ? detailRow("Probe Error", technical.error) : ""}
      </div>
    </section>

    <section class="details-section">
      <h3>Tags</h3>
      <div class="tag-list">${tags}<span class="tag add">+</span></div>
    </section>

    <section class="details-section">
      <h3>TMDB Info</h3>
      <div class="detail-list">
        ${detailRow("ID", tmdb.id || "Unknown")}
        ${detailRow("Popularity", tmdb.popularity || "Unknown")}
        ${detailRow("Vote Average", tmdb.voteAverage || "Unknown")}
        ${detailRow("Vote Count", tmdb.voteCount || "Unknown")}
      </div>
    </section>
  `;
  detailsPanel.querySelector(".details-poster")?.style.setProperty("--poster", movie.poster);
  detailsPanel.querySelector(".delete-movie-button")?.addEventListener("click", () => deleteMovie(movie));
}

function detailRow(label, value, raw = false) {
  const rendered = raw ? value : escapeHTML(String(value || ""));
  return `<div class="detail-row"><span>${escapeHTML(label)}</span><strong>${rendered}</strong></div>`;
}

function mediaTechnicalSummary(metadata = {}) {
  const videoParts = [];
  if (metadata.video_codec) {
    videoParts.push(metadata.video_codec.toUpperCase());
  }
  if (metadata.width && metadata.height) {
    videoParts.push(`${metadata.width}x${metadata.height}`);
  }
  if (metadata.frame_rate) {
    videoParts.push(`${Number(metadata.frame_rate).toFixed(3).replace(/\.?0+$/, "")} fps`);
  }

  return {
    fileSize: formatBytes(metadata.file_size),
    video: videoParts.join(" / ") || "Not analyzed",
    audio: formatAudioTracks(metadata.audio_tracks, metadata.audio_codec),
    subtitles: formatSubtitleTracks(metadata.subtitle_tracks),
    error: metadata.probe_error || "",
  };
}

function formatAudioTracks(tracks = [], fallbackCodec = "") {
  if (!Array.isArray(tracks) || tracks.length === 0) {
    return fallbackCodec ? fallbackCodec.toUpperCase() : "Not analyzed";
  }
  return tracks
    .map((track) => {
      const parts = [];
      if (track.codec) {
        parts.push(track.codec.toUpperCase());
      }
      if (track.channels) {
        parts.push(`${track.channels}ch`);
      }
      if (track.language) {
        parts.push(track.language);
      }
      return parts.join(" / ");
    })
    .filter(Boolean)
    .join(", ") || "Not analyzed";
}

function formatSubtitleTracks(tracks = []) {
  if (!Array.isArray(tracks) || tracks.length === 0) {
    return "None detected";
  }
  return tracks
    .map((track) => {
      const parts = [];
      if (track.codec) {
        parts.push(track.codec.toUpperCase());
      }
      if (track.language) {
        parts.push(track.language);
      }
      return parts.join(" / ");
    })
    .filter(Boolean)
    .join(", ") || "None detected";
}

function formatBytes(bytes) {
  const value = Number(bytes || 0);
  if (!value) {
    return "Not analyzed";
  }
  const units = ["B", "KB", "MB", "GB", "TB"];
  const exponent = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1);
  const amount = value / 1024 ** exponent;
  return `${amount.toFixed(amount >= 10 || exponent === 0 ? 0 : 1)} ${units[exponent]}`;
}

function escapeHTML(value) {
  return String(value).replace(/[&<>"']/g, (char) => {
    const entities = {
      "&": "&amp;",
      "<": "&lt;",
      ">": "&gt;",
      '"': "&quot;",
      "'": "&#039;",
    };
    return entities[char];
  });
}

function escapeAttribute(value) {
  return escapeHTML(value).replaceAll("`", "&#096;");
}

function formatDate(value) {
  if (!value) {
    return "Not recorded";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return date.toLocaleString();
}

async function scanLibrary() {
  scanButton.disabled = true;
  scanStatus.textContent = "Scanning media folders...";
  startScanProgress();

  try {
    const response = await fetch("/api/scan/movies", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ force_metadata: forceMetadataRequested() }),
    });

    const payload = await readJSONResponse(response);
    if (!response.ok) {
      throw new Error(payload.error || "Scan failed");
    }

    movies = reconcileMovieStatuses((payload.movies || []).map(movieFromRecord));
    updateMovieFilterOptions();
    moviesLoaded = true;
    moviePage = 1;
    selectedMovie = movies[0] || null;
    await loadMetadataFacets();

    if (movies.length > 0) {
      setScanStatusWithErrors(
        scanStatus,
        `Scan complete: ${scanCounterSummary(payload)}.`,
        payload.errors,
        "Movie scan issues"
      );
    } else {
      const hasErrors = normalizeScanErrors(payload.errors).length > 0;
      setScanStatusWithErrors(
        scanStatus,
        hasErrors ? "No movies saved." : `No movie files found in ${payload.media_path}.`,
        payload.errors,
        "Movie scan issues"
      );
    }

    render();
  } catch (error) {
    scanStatus.textContent = error.message;
  } finally {
    scanButton.disabled = false;
    stopScanProgress();
  }
}

async function loadSettings() {
  try {
    const response = await fetch("/api/settings");
    const settings = await response.json();
    if (!response.ok) {
      throw new Error(settings.error || "Could not load settings");
    }

    mediaSources =
      settings.media_sources && settings.media_sources.length > 0
        ? settings.media_sources
        : [
            { path: settings.media_path || "data/movies", content_type: "movie" },
            { path: settings.tv_media_path || "data/tvshows", content_type: "tv" },
          ];
    assetSources =
      settings.asset_sources && settings.asset_sources.length > 0
        ? settings.asset_sources
        : [{ path: "data/assets", scope: "global" }];
    streamingMode = settings.streaming_mode === "hls" ? "hls" : "direct";
    renderSources();
    renderAssetSources();
    renderStreamingMode();
    setSettingsDirty(false);
    setSettingsStatus("All changes saved", "saved");
    await loadAssets();
  } catch (error) {
    setSettingsStatus(error.message);
  }
}

function renderStreamingMode() {
  if (!streamingModeControl) {
    return;
  }
  streamingModeControl.querySelectorAll('input[name="streaming-mode"]').forEach((input) => {
    input.checked = input.value === streamingMode;
  });
}

function renderSources() {
  if (!sourceList) {
    return;
  }

  if (mediaSources.length === 0) {
    const empty = document.createElement("section");
    empty.className = "source-empty";
    empty.innerHTML = `<h2>No media sources</h2><p>Add a movie or TV source folder to start scanning.</p>`;
    sourceList.replaceChildren(empty);
    return;
  }

  sourceList.replaceChildren(...mediaSources.map(sourceRow));
}

function renderAssetSources() {
  if (!assetSourceList) {
    return;
  }

  if (assetSources.length === 0) {
    const empty = document.createElement("section");
    empty.className = "source-empty";
    empty.innerHTML = `<h2>No asset sources</h2><p>Add a folder for channel bumpers, commercials, promos, and station IDs.</p>`;
    assetSourceList.replaceChildren(empty);
    return;
  }

  assetSourceList.replaceChildren(...assetSources.map(assetSourceRow));
}

function renderAssetSummary() {
  if (!assetSummary) {
    return;
  }
  if (channelAssets.length === 0) {
    assetSummary.innerHTML = `<p>No channel assets scanned yet.</p>`;
    return;
  }
  const counts = channelAssets.reduce((summary, asset) => {
    summary[asset.asset_type] = (summary[asset.asset_type] || 0) + 1;
    return summary;
  }, {});
  const labels = [
    ["bumper", "Bumpers"],
    ["commercial", "Commercials"],
    ["promo", "Promos"],
    ["station_id", "Station IDs"],
  ];
  assetSummary.innerHTML = `
    <div class="asset-counts">
      ${labels
        .map(([key, label]) => `<span><strong>${counts[key] || 0}</strong>${label}</span>`)
        .join("")}
    </div>
  `;
}

function sourceRow(source, index) {
  const row = document.createElement("article");
  row.className = "source-row";
  row.innerHTML = `
    <select aria-label="Source content type">
      <option value="movie">Movies</option>
      <option value="tv">TV Shows</option>
    </select>
    <input type="text" aria-label="Source path" placeholder="/media/movies" />
    <button class="button ghost compact scan-source" type="button">Scan</button>
    <button class="icon-button delete-source" type="button" aria-label="Remove source">
      <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M3 6h18M8 6V4h8v2M6 6l1 15h10l1-15" /></svg>
    </button>
  `;

  const [typeSelect] = row.querySelectorAll("select");
  const pathInput = row.querySelector("input");
  typeSelect.value = source.content_type || "movie";
  pathInput.value = source.path || "";

  typeSelect.addEventListener("change", () => {
    mediaSources[index] = { ...mediaSources[index], content_type: typeSelect.value };
    setSettingsDirty(true);
  });
  pathInput.addEventListener("input", () => {
    mediaSources[index] = { ...mediaSources[index], path: pathInput.value };
    setSettingsDirty(true);
  });
  row.querySelector(".delete-source").addEventListener("click", () => {
    mediaSources.splice(index, 1);
    renderSources();
    setSettingsDirty(true);
  });
  row.querySelector(".scan-source").addEventListener("click", async () => {
    syncSourcesFromDOM();
    const sourceToScan = { ...mediaSources[index] };
    if (!(await saveSettings())) {
      return;
    }
    await scanSource(sourceToScan);
  });
  return row;
}

function assetSourceRow(source, index) {
  const row = document.createElement("article");
  row.className = "source-row asset-source-row";
  row.innerHTML = `
    <select aria-label="Asset source scope">
      <option value="global">Global</option>
    </select>
    <input type="text" aria-label="Asset source path" placeholder="/media/retrotv-assets" />
    <button class="icon-button delete-source" type="button" aria-label="Remove asset source">
      <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M3 6h18M8 6V4h8v2M6 6l1 15h10l1-15" /></svg>
    </button>
  `;

  const scopeSelect = row.querySelector("select");
  const pathInput = row.querySelector("input");
  scopeSelect.value = source.scope || "global";
  pathInput.value = source.path || "";

  scopeSelect.addEventListener("change", () => {
    assetSources[index] = { ...assetSources[index], scope: scopeSelect.value };
    setSettingsDirty(true);
  });
  pathInput.addEventListener("input", () => {
    assetSources[index] = { ...assetSources[index], path: pathInput.value };
    setSettingsDirty(true);
  });
  row.querySelector(".delete-source").addEventListener("click", () => {
    assetSources.splice(index, 1);
    renderAssetSources();
    setSettingsDirty(true);
  });
  return row;
}

function addSource() {
  mediaSources.push({ path: "", content_type: "movie" });
  renderSources();
  setSettingsDirty(true);
}

function addAssetSource() {
  assetSources.push({ path: "", scope: "global" });
  renderAssetSources();
  setSettingsDirty(true);
}

function syncSourcesFromDOM() {
  if (!sourceList) {
    return;
  }

  mediaSources = [...sourceList.querySelectorAll(".source-row")].map((row) => ({
    content_type: row.querySelector("select").value,
    path: row.querySelector("input").value.trim(),
  }));
}

function syncAssetSourcesFromDOM() {
  if (!assetSourceList) {
    return;
  }

  assetSources = [...assetSourceList.querySelectorAll(".asset-source-row")].map((row) => ({
    scope: row.querySelector("select").value,
    path: row.querySelector("input").value.trim(),
  }));
}

function sourcePayload() {
  syncSourcesFromDOM();
  syncAssetSourcesFromDOM();
  const sources = mediaSources.filter((source) => source.path);
  const assets = assetSources.filter((source) => source.path);
  return {
    media_sources: sources,
    asset_sources: assets,
    media_path: sources.find((source) => source.content_type === "movie")?.path || "",
    tv_media_path: sources.find((source) => source.content_type === "tv")?.path || "",
    streaming_mode: streamingMode,
  };
}

function setSettingsStatus(message, state = "") {
  if (settingsStatus) {
    settingsStatus.textContent = message;
    settingsStatus.dataset.state = state;
  }
}

function setSettingsDirty(dirty) {
  settingsDirty = dirty;
  if (saveSettingsButton) {
    saveSettingsButton.disabled = !dirty;
  }
  if (dirty) {
    setSettingsStatus("Unsaved changes", "dirty");
  }
}

function setSettingsSaving(saving) {
  if (saveSettingsButton) {
    saveSettingsButton.disabled = saving || !settingsDirty;
  }
  if (saveSettingsLabel) {
    saveSettingsLabel.textContent = saving ? "Saving…" : "Save";
  }
  if (saveSettingsSpinner) {
    saveSettingsSpinner.hidden = !saving;
  }
}

function setScanStatusWithErrors(element, message, errors, title) {
  if (!element) {
    return;
  }
  const details = normalizeScanErrors(errors);
  if (details.length === 0) {
    element.textContent = message;
    return;
  }

  const summary = summarizeScanErrors(details);
  const text = document.createElement("span");
  text.textContent = `${message} ${details.length} issue${details.length === 1 ? "" : "s"} need attention`;

  const button = document.createElement("button");
  button.className = "inline-status-button";
  button.type = "button";
  button.textContent = "View issues";
  button.addEventListener("click", () => openScanErrorModal(title, details));

  element.replaceChildren(text, button);
  if (summary) {
    const summaryText = document.createElement("span");
    summaryText.className = "scan-error-status-summary";
    summaryText.textContent = summary;
    element.appendChild(summaryText);
  }
}

function normalizeScanErrors(errors) {
  return Array.isArray(errors) ? errors.filter(Boolean).map(String) : [];
}

function summarizeScanErrors(errors) {
  const groups = groupScanErrors(errors);
  return groups
    .slice(0, 3)
    .map((group) => `${group.items.length} ${group.title.toLowerCase()}`)
    .join(", ");
}

function groupScanErrors(errors) {
  const groups = new Map();
  normalizeScanErrors(errors).forEach((error) => {
    const category = scanErrorCategory(error);
    if (!groups.has(category.title)) {
      groups.set(category.title, { ...category, items: [] });
    }
    groups.get(category.title).items.push(parseScanError(error));
  });
  return [...groups.values()];
}

function scanErrorCategory(error) {
  const lower = error.toLowerCase();
  if (lower.includes("database is locked") || lower.includes("sqlite_busy")) {
    return {
      title: "Database busy",
      help: "The database was locked while the scanner was saving. Re-run the scan after the current scan or schedule work finishes.",
    };
  }
  if (lower.includes("year is required")) {
    return {
      title: "Needs manual match",
      help: "These folders do not include a year, so the scanner saved what it could and needs you to refresh metadata from the item detail page.",
    };
  }
  if (lower.includes("404 not found")) {
    return {
      title: "Episode metadata not found",
      help: "TMDB did not have metadata for that specific episode. The show can still exist locally; refresh metadata if the match is wrong.",
    };
  }
  if (lower.includes("tmdb_read_token") || lower.includes("tmdb token")) {
    return {
      title: "TMDB token missing",
      help: "Set TMDB_READ_TOKEN in your development environment before refreshing metadata or scanning new items.",
    };
  }
  if (lower.includes("tmdb request failed")) {
    return {
      title: "TMDB request failed",
      help: "The scanner could not get metadata from TMDB for these paths. Re-run later or refresh metadata on the item.",
    };
  }
  return {
    title: "Other scan issues",
    help: "These need a closer look. The original scanner message is shown for each path.",
  };
}

function parseScanError(error) {
  const separator = error.indexOf(": ");
  if (separator <= 0) {
    return { path: "", detail: error };
  }
  return {
    path: error.slice(0, separator),
    detail: error.slice(separator + 2),
  };
}

function openScanErrorModal(title, errors) {
  if (!scanErrorModal || !scanErrorList) {
    return;
  }
  const details = normalizeScanErrors(errors);
  scanErrorTitle.textContent = title || "Scan issues";
  scanErrorSummaryText.textContent = `${details.length} issue${details.length === 1 ? "" : "s"} need attention.`;

  const groups = groupScanErrors(details).map((group) => {
    const section = document.createElement("section");
    section.className = "scan-error-group";

    const heading = document.createElement("h3");
    heading.textContent = group.title;
    const count = document.createElement("span");
    count.textContent = `${group.items.length}`;
    heading.appendChild(count);

    const help = document.createElement("p");
    help.textContent = group.help;

    const list = document.createElement("ul");
    group.items.forEach((item) => {
      const row = document.createElement("li");
      if (item.path) {
        const path = document.createElement("strong");
        path.className = "scan-error-path";
        path.textContent = item.path;
        row.append(path, document.createTextNode(item.detail ? `: ${item.detail}` : ""));
      } else {
        row.textContent = item.detail;
      }
      list.appendChild(row);
    });

    section.append(heading, help, list);
    return section;
  });

  scanErrorList.replaceChildren(...groups);
  scanErrorModal.hidden = false;
  scanErrorModal.classList.add("open");
  scanErrorModal.setAttribute("aria-hidden", "false");
}

function closeScanErrorModal() {
  if (!scanErrorModal) {
    return;
  }
  scanErrorModal.classList.remove("open");
  scanErrorModal.setAttribute("aria-hidden", "true");
  scanErrorModal.hidden = true;
}

function startScanProgress() {
  if (!scanProgressBar) {
    return;
  }
  window.clearTimeout(scanProgressHideTimer);
  scanProgressHideTimer = null;
  scanProgressBar.hidden = false;
  renderScanProgress({ active: true, phase: "Starting scan", current: "Preparing folders..." });
  if (!scanProgressTimer) {
    scanProgressTimer = window.setInterval(loadScanProgress, 700);
  }
  loadScanProgress();
}

function stopScanProgress() {
  if (!scanProgressBar) {
    return;
  }
  window.clearInterval(scanProgressTimer);
  scanProgressTimer = null;
  loadScanProgress({ force: true });
  scanProgressHideTimer = window.setTimeout(() => {
    scanProgressBar.hidden = true;
    scanProgressHideTimer = null;
  }, 3500);
}

async function loadScanProgress({ force = false } = {}) {
  if (scanProgressController) {
    if (!force) {
      return;
    }
    scanProgressController.abort();
  }

  const controller = new AbortController();
  scanProgressController = controller;
  try {
    const response = await fetch("/api/scan/status", { signal: controller.signal });
    const status = await response.json();
    if (!response.ok) {
      return;
    }
    renderScanProgress(status);
  } catch {
    // Progress is best-effort; the scan request itself reports real failures.
  } finally {
    if (scanProgressController === controller) {
      scanProgressController = null;
    }
  }
}

function renderScanProgress(status) {
  if (!scanProgressBar || !status) {
    return;
  }

  const phase = status.phase || (status.active ? "Scanning" : "Scan complete");
  const current = status.current || status.source || (status.active ? "Working..." : "Finished");
  scanProgressPhase.textContent = phase;
  scanProgressCurrent.textContent =
    status.source && status.current && status.source !== status.current
      ? `${current} • ${status.source}`
      : current;

  if (status.total_items > 0) {
    scanProgressCount.textContent = `${status.current_item || 0} / ${status.total_items}`;
    scanProgressCount.hidden = false;
  } else {
    scanProgressCount.hidden = true;
  }

  if (status.errors > 0) {
    scanProgressErrors.textContent = `${status.errors} errors`;
    if (status.error_details && status.error_details.length > 0) {
      scanProgressErrors.title = status.error_details.join("\n");
    } else {
      scanProgressErrors.removeAttribute("title");
    }
    scanProgressErrors.hidden = false;
  } else {
    scanProgressErrors.removeAttribute("title");
    scanProgressErrors.hidden = true;
  }
}

async function loadMovies() {
  try {
    const response = await fetch("/api/movies");
    const payload = await response.json();
    if (!response.ok) {
      throw new Error(payload.error || "Could not load movies");
    }

    movies = reconcileMovieStatuses((payload.movies || []).map(movieFromRecord));
    updateMovieFilterOptions();
    moviesLoaded = true;
    moviePage = 1;
    selectedMovie = movies[0] || null;
    if (movies.length > 0) {
      scanStatus.textContent = `Loaded ${movies.length.toLocaleString()} movies from the database.`;
    }
    render();
    setupRuleBuilder();
  } catch (error) {
    scanStatus.textContent = error.message;
    render();
    setupRuleBuilder();
  } finally {
    renderDashboard();
    if (channels.length > 0) loadDashboardNowPlaying();
  }
}

async function deleteMovie(movie) {
  if (!movie?.id) {
    return;
  }
  if (!window.confirm(`Delete "${movie.title}" from the RetroTV database?`)) {
    return;
  }

  scanStatus.textContent = `Deleting ${movie.title}...`;
  try {
    const response = await fetch(`/api/movies/${movie.id}`, { method: "DELETE" });
    if (!response.ok) {
      const payload = await response.json();
      throw new Error(payload.error || "Could not delete movie");
    }

    movies = reconcileMovieStatuses(movies.filter((existing) => existing.id !== movie.id));
    moviePage = clampPage(moviePage, pageCount(movies.length));
    selectedMovie = movies[0] || null;
    scanStatus.textContent = `Deleted ${movie.title}. Channel schedules were refreshed.`;
    render();
    await loadMetadataFacets();
  } catch (error) {
    scanStatus.textContent = error.message;
  }
}

async function refreshMovieMetadata(movie) {
  if (!movie?.id) {
    return;
  }

  const searchTitle = movie.searchTitle || movie.title;
  scanStatus.textContent = `Finding metadata matches for ${searchTitle}...`;
  try {
    const matchesURL = `/api/movies/${movie.id}/metadata-matches`;
    const matchesResponse = await fetch(`${matchesURL}?query=${encodeURIComponent(searchTitle)}`);
    const matchesPayload = await readJSONResponse(matchesResponse);
    if (!matchesResponse.ok) {
      throw new Error(matchesPayload.error || "Could not search movie metadata");
    }
    const match = await chooseMetadataMatch(searchTitle, matchesPayload.matches || [], "Movie", matchesURL);
    if (!match) {
      scanStatus.textContent = "Metadata refresh canceled.";
      return;
    }

    scanStatus.textContent = `Refreshing metadata for ${movie.title}...`;
    const response = await fetch(`/api/movies/${movie.id}/refresh-metadata`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ tmdb_id: match.tmdb_id }),
    });
    const payload = await readJSONResponse(response);
    if (!response.ok) {
      throw new Error(payload.error || "Could not refresh movie metadata");
    }

    const refreshed = movieFromRecord(payload, movies.findIndex((existing) => existing.id === movie.id));
    movies = reconcileMovieStatuses(movies.map((existing) => (existing.id === movie.id ? refreshed : existing)));
    selectedMovie = refreshed;
    scanStatus.textContent = `Refreshed metadata for ${refreshed.title}.`;
    render();
    await loadMetadataFacets();
  } catch (error) {
    scanStatus.textContent = error.message;
  }
}

async function loadTVShows() {
  try {
    const response = await fetch("/api/tvshows");
    const payload = await response.json();
    if (!response.ok) {
      throw new Error(payload.error || "Could not load TV shows");
    }

    tvShows = reconcileTVShowStatuses((payload.tv_shows || []).map(tvShowFromRecord));
    updateTVFilterOptions();
    tvShowsLoaded = true;
    tvShowPage = 1;
    selectedTVShow = tvShows[0] || null;
    renderTVShows();
    setupRuleBuilder();
  } catch (error) {
    tvScanStatus.textContent = error.message;
    renderTVShows();
    setupRuleBuilder();
  } finally {
    renderDashboard();
  }
}

async function deleteTVShow(show) {
  if (!show?.id) {
    return;
  }
  if (!window.confirm(`Delete "${show.title}" from the RetroTV database?`)) {
    return;
  }

  tvScanStatus.textContent = `Deleting ${show.title}...`;
  try {
    const response = await fetch(`/api/tvshows/${show.id}`, { method: "DELETE" });
    if (!response.ok) {
      const payload = await response.json();
      throw new Error(payload.error || "Could not delete TV show");
    }

    tvShows = reconcileTVShowStatuses(tvShows.filter((existing) => existing.id !== show.id));
    updateTVFilterOptions();
    tvShowPage = clampPage(tvShowPage, pageCount(tvShows.length));
    selectedTVShow = tvShows[0] || null;
    tvScanStatus.textContent = `Deleted ${show.title}.`;
    renderTVShows();
    setupRuleBuilder();
  } catch (error) {
    tvScanStatus.textContent = error.message;
  }
}

async function refreshTVShowMetadata(show) {
  if (!show?.id) {
    return;
  }

  const searchTitle = show.searchTitle || show.title;
  tvScanStatus.textContent = `Finding metadata matches for ${searchTitle}...`;
  try {
    const matchesURL = `/api/tvshows/${show.id}/metadata-matches`;
    const matchesResponse = await fetch(`${matchesURL}?query=${encodeURIComponent(searchTitle)}`);
    const matchesPayload = await readJSONResponse(matchesResponse);
    if (!matchesResponse.ok) {
      throw new Error(matchesPayload.error || "Could not search TV metadata");
    }
    const match = await chooseMetadataMatch(searchTitle, matchesPayload.matches || [], "TV Show", matchesURL);
    if (!match) {
      tvScanStatus.textContent = "Metadata refresh canceled.";
      return;
    }

    tvScanStatus.textContent = `Refreshing metadata for ${show.title}...`;
    const response = await fetch(`/api/tvshows/${show.id}/refresh-metadata`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ tmdb_id: match.tmdb_id }),
    });
    const payload = await readJSONResponse(response);
    if (!response.ok) {
      throw new Error(payload.error || "Could not refresh TV metadata");
    }

    const refreshed = tvShowFromRecord(payload, tvShows.findIndex((existing) => existing.id === show.id));
    tvShows = tvShows.map((existing) => (existing.id === show.id ? refreshed : existing));
    selectedTVShow = refreshed;
    tvScanStatus.textContent = `Refreshed metadata for ${refreshed.title}.`;
    renderTVShows();
    setupRuleBuilder();
    await loadMetadataFacets();
  } catch (error) {
    tvScanStatus.textContent = error.message;
  }
}

function chooseMetadataMatch(title, matches, mediaType = "Movie", matchesURL = "") {
  matches = Array.isArray(matches) ? matches : [];

  const modal = document.querySelector("#metadata-match-modal");
  const list = document.querySelector("#metadata-match-list");
  const summary = document.querySelector("#metadata-match-summary");
  const closeButton = document.querySelector("#metadata-match-close");
  const cancelButton = document.querySelector("#metadata-match-cancel");
  const searchForm = document.querySelector("#metadata-match-search");
  const searchInput = document.querySelector("#metadata-match-query");
  const searchButton = document.querySelector("#metadata-match-search-button");
  let visibleMatches = matches.slice(0, 10);

  const renderMatches = (results, query) => {
    visibleMatches = results.slice(0, 10);
    summary.innerHTML = `Showing top ${visibleMatches.length} results for <strong>“${escapeHTML(query)}”</strong>`;
    list.innerHTML = visibleMatches.length
      ? visibleMatches.map((match, index) => metadataMatchCard(match, index, mediaType)).join("")
      : `<div class="metadata-match-empty"><strong>No matches found</strong><span>Try a different title or remove the year.</span></div>`;
    list.scrollTop = 0;
  };

  searchInput.value = title;
  renderMatches(visibleMatches, title);
  modal.hidden = false;
  modal.classList.add("open");
  modal.setAttribute("aria-hidden", "false");
  document.body.classList.add("modal-open");
  searchInput.focus();
  searchInput.select();

  return new Promise((resolve) => {
    const finish = (match = null) => {
      modal.classList.remove("open");
      modal.hidden = true;
      modal.setAttribute("aria-hidden", "true");
      document.body.classList.remove("modal-open");
      modal.removeEventListener("click", handleClick);
      searchForm.removeEventListener("submit", handleSearch);
      document.removeEventListener("keydown", handleKeydown);
      resolve(match);
    };
    const handleClick = (event) => {
      const selectButton = event.target.closest("[data-metadata-match]");
      if (selectButton) {
        finish(visibleMatches[Number(selectButton.dataset.metadataMatch)]);
      } else if (event.target === modal || event.target === closeButton || closeButton.contains(event.target) || event.target === cancelButton) {
        finish();
      }
    };
    const handleKeydown = (event) => {
      if (event.key === "Escape") {
        finish();
      }
    };
    const handleSearch = async (event) => {
      event.preventDefault();
      const query = searchInput.value.trim();
      if (!query || !matchesURL) {
        return;
      }
      searchButton.disabled = true;
      searchButton.textContent = "Searching…";
      list.innerHTML = `<div class="metadata-match-empty"><span>Searching TMDB…</span></div>`;
      try {
        const response = await fetch(`${matchesURL}?query=${encodeURIComponent(query)}`);
        const payload = await readJSONResponse(response);
        if (!response.ok) {
          throw new Error(payload.error || "Could not search TMDB");
        }
        renderMatches(payload.matches || [], query);
      } catch (error) {
        list.innerHTML = `<div class="metadata-match-empty error"><strong>Search failed</strong><span>${escapeHTML(error.message)}</span></div>`;
      } finally {
        searchButton.disabled = false;
        searchButton.textContent = "Search";
      }
    };
    modal.addEventListener("click", handleClick);
    searchForm.addEventListener("submit", handleSearch);
    document.addEventListener("keydown", handleKeydown);
  });
}

function metadataMatchCard(match, index, mediaType) {
  const year = match.year || (match.release_date || "").slice(0, 4) || "Unknown";
  const rating = Number(match.vote_average || 0);
  const ratingClass = rating && rating < 5 ? "low" : "";
  const poster = match.poster_path
    ? `<img src="https://image.tmdb.org/t/p/w185${escapeAttribute(match.poster_path)}" alt="" loading="lazy" />`
    : `<span class="metadata-match-poster-fallback" aria-hidden="true"><svg viewBox="0 0 24 24"><path d="M4 5h16v14H4zM8 5v14M16 5v14M4 10h4M16 10h4M4 15h4M16 15h4" /></svg></span>`;
  const overview = match.overview || "No description is available for this result.";

  return `
    <article class="metadata-match-card">
      <div class="metadata-match-poster">${poster}</div>
      <div class="metadata-match-copy">
        <h3>${escapeHTML(match.title || "Untitled")}</h3>
        <p class="metadata-match-type">${escapeHTML(mediaType)}</p>
        <p class="metadata-match-overview">${escapeHTML(overview)}</p>
      </div>
      <div class="metadata-match-facts">
        <strong>${escapeHTML(String(year))}</strong>
        ${rating ? `<span class="metadata-match-rating ${ratingClass}">TMDB ${rating.toFixed(1)}</span>` : ""}
      </div>
      <button class="button primary metadata-match-select" type="button" data-metadata-match="${index}">Select</button>
    </article>`;
}

function metadataMatchLabel(match) {
  const year = match.year || (match.release_date || "").slice(0, 4) || "Unknown";
  return `${match.title || "Untitled"} (${year}) - TMDB ${match.tmdb_id}`;
}

async function saveSettings() {
  setSettingsSaving(true);
  setSettingsStatus("Saving settings…", "saving");

  try {
    const response = await fetch("/api/settings", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(sourcePayload()),
    });

    const settings = await response.json();
    if (!response.ok) {
      throw new Error(settings.error || "Could not save settings");
    }

    mediaSources = settings.media_sources || [];
    assetSources = settings.asset_sources || [];
    streamingMode = settings.streaming_mode === "hls" ? "hls" : "direct";
    renderSources();
    renderAssetSources();
    renderStreamingMode();
    setSettingsDirty(false);
    setSettingsStatus("Saved", "saved");
    return true;
  } catch (error) {
    setSettingsStatus(error.message, "error");
    return false;
  } finally {
    setSettingsSaving(false);
  }
}

async function loadAssets() {
  try {
    const response = await fetch("/api/assets");
    const payload = await response.json();
    if (!response.ok) {
      throw new Error(payload.error || "Could not load channel assets");
    }
    channelAssets = payload.assets || [];
    renderAssetSummary();
  } catch (error) {
    channelAssets = [];
    renderAssetSummary();
    setSettingsStatus(error.message);
  }
}

async function scanAssets() {
  if (!scanAssetsButton) {
    return;
  }
  scanAssetsButton.disabled = true;
  setSettingsStatus("Scanning channel assets...");
  startScanProgress();
  try {
    if (!(await saveSettings())) {
      return;
    }
    const response = await fetch("/api/assets/scan", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ force_metadata: forceMetadataRequested() }),
    });
    const payload = await readJSONResponse(response);
    if (!response.ok) {
      throw new Error(payload.error || "Asset scan failed");
    }
    if (Array.isArray(payload.asset_sources)) {
      assetSources = payload.asset_sources;
      renderAssetSources();
    }
    channelAssets = payload.assets || [];
    renderAssetSummary();
    setScanStatusWithErrors(
      settingsStatus,
      `Scanned ${payload.scanned} assets; saved ${payload.count}.`,
      payload.errors,
      "Asset scan issues"
    );
  } catch (error) {
    setSettingsStatus(error.message);
  } finally {
    stopScanProgress();
    scanAssetsButton.disabled = false;
  }
}

async function scanTVShows() {
  scanTVButton.disabled = true;
  tvScanStatus.textContent = "Scanning TV show folders...";
  startScanProgress();

  try {
    const response = await fetch("/api/scan/tvshows", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({}),
    });

    const payload = await readJSONResponse(response);
    if (!response.ok) {
      throw new Error(payload.error || "TV scan failed");
    }

    tvShows = reconcileTVShowStatuses((payload.tv_shows || []).map(tvShowFromRecord));
    updateTVFilterOptions();
    tvShowsLoaded = true;
    tvShowPage = 1;
    selectedTVShow = tvShows[0] || null;
    if (tvShows.length > 0) {
      setScanStatusWithErrors(
        tvScanStatus,
        `Scan complete: ${scanCounterSummary(payload)}.`,
        payload.errors,
        "TV scan issues"
      );
    } else {
      const hasErrors = normalizeScanErrors(payload.errors).length > 0;
      setScanStatusWithErrors(
        tvScanStatus,
        hasErrors ? "No TV shows saved." : `No TV episodes found in ${payload.tv_media_path}.`,
        payload.errors,
        "TV scan issues"
      );
    }
    renderTVShows();
    setupRuleBuilder();
  } catch (error) {
    tvScanStatus.textContent = error.message;
  } finally {
    scanTVButton.disabled = false;
    stopScanProgress();
  }
}

async function scanSource(source) {
  if (!source || !source.path) {
    setSettingsStatus("Source path is required.");
    return;
  }

  setSettingsStatus(`Scanning ${source.path}...`);
  startScanProgress();
  try {
    const isTV = source.content_type === "tv";
    const response = await fetch(isTV ? "/api/scan/tvshows" : "/api/scan/movies", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(isTV
        ? { tv_media_path: source.path, force_metadata: forceMetadataRequested() }
        : { media_path: source.path, force_metadata: forceMetadataRequested() }),
    });
    const payload = await readJSONResponse(response);
    if (!response.ok) {
      throw new Error(payload.error || "Scan failed");
    }

    if (isTV) {
      tvShows = reconcileTVShowStatuses((payload.tv_shows || []).map(tvShowFromRecord));
      updateTVFilterOptions();
      tvShowsLoaded = true;
      tvShowPage = 1;
      selectedTVShow = tvShows[0] || selectedTVShow;
      renderTVShows();
      setupRuleBuilder();
    } else {
      movies = reconcileMovieStatuses((payload.movies || []).map(movieFromRecord));
      moviesLoaded = true;
      moviePage = 1;
      selectedMovie = movies[0] || selectedMovie;
      render();
      await loadMetadataFacets();
    }
    setScanStatusWithErrors(
      settingsStatus,
      `Scan complete: ${scanCounterSummary(payload)}.`,
      payload.errors,
      isTV ? "TV scan issues" : "Movie scan issues"
    );
  } catch (error) {
    setSettingsStatus(error.message);
  } finally {
    stopScanProgress();
  }
}

async function scanAllSources() {
  scanAllSourcesButton.disabled = true;
  setSettingsStatus("Saving sources and scanning all media...");
  startScanProgress();
  try {
    if (!(await saveSettings())) {
      return;
    }
    await scanLibrary();
    await scanTVShows();
    setSettingsStatus("Finished scanning all configured sources.");
  } finally {
    scanAllSourcesButton.disabled = false;
    stopScanProgress();
  }
}

function forceMetadataRequested() {
  return Boolean(forceMetadataScan?.checked);
}

function scanCounterSummary(payload = {}) {
  return `${payload.discovered || 0} discovered, ${payload.unchanged || 0} unchanged, ${payload.new || 0} new, ${payload.changed || 0} changed, ${payload.missing || 0} missing`;
}

function movieFromRecord(movie, index) {
  movie = movie || {};
  index = Math.max(0, Number(index) || 0);
  const title = movie.title || movie.local_title || "Unknown Movie";
  const year = movie.year || movie.local_year || "";
  const posterURL = artworkURL(movie.poster_local_path);
  const fallbackPoster = [
    "linear-gradient(145deg, #141a24 0%, #5b2b2b 50%, #0a0d12 100%)",
    "linear-gradient(145deg, #18241f 0%, #2d6a4f 50%, #080c0a 100%)",
    "linear-gradient(145deg, #161623 0%, #5e548e 50%, #07070d 100%)",
    "linear-gradient(145deg, #22160f 0%, #b36b21 50%, #090605 100%)",
  ][index % 4];

  return {
    id: movie.id,
    title,
    searchTitle: movie.local_title || title,
    year,
    originalTitle: movie.original_title,
    runtime: movie.runtime_minutes ? `${movie.runtime_minutes}m` : "Unknown",
    rating: movie.rating || "NR",
    ratingValue: movie.rating || "",
    studio: movie.studio || "",
    genres: Array.isArray(movie.genres) ? movie.genres.filter(Boolean) : [],
    actors: Array.isArray(movie.actors) ? movie.actors.filter(Boolean) : [],
    directors: Array.isArray(movie.directors) ? movie.directors.filter(Boolean) : [],
    writers: Array.isArray(movie.writers) ? movie.writers.filter(Boolean) : [],
    countries: Array.isArray(movie.countries) ? movie.countries.filter(Boolean) : [],
    languages: Array.isArray(movie.languages) ? movie.languages.filter(Boolean) : [],
    releaseDate: movie.release_date || "",
    runtimeMinutes: movie.runtime_minutes || "",
    source: movie.tmdb_id ? "The Movie DB" : "Local scan",
    filePath: movie.file_path,
    tmdbID: Number(movie.tmdb_id || 0),
    localYear: Number(movie.local_year || 0),
    lastSeenAt: movie.last_seen_at || "",
    missingSince: movie.missing_since || "",
    missing: Boolean(movie.missing_since),
    mediaMetadata: movie.media_metadata || {},
    overview: movie.overview || "Metadata is not available for this movie.",
    tags: Array.isArray(movie.tags) ? movie.tags.filter(Boolean) : [],
    added: movie.created_at,
    modified: movie.updated_at,
    tmdb: {
      id: movie.tmdb_id || "Unknown",
      popularity: "Unknown",
      voteAverage: movie.vote_average || "Unknown",
      voteCount: "Unknown",
    },
    poster: posterURL ? `url("${posterURL}") center / cover` : fallbackPoster,
  };
}

function reconcileMovieStatuses(movieRecords) {
  const identityGroups = new Map();
  movieRecords.forEach((movie) => {
    const normalizedTitle = String(movie.searchTitle || movie.title).toLowerCase().replace(/[^a-z0-9]+/g, "");
    const keys = [
      normalizedTitle ? `title:${normalizedTitle}` : "",
      movie.tmdbID > 0 ? `tmdb:${movie.tmdbID}` : "",
    ].filter(Boolean);
    keys.forEach((key) => identityGroups.set(key, [...(identityGroups.get(key) || []), movie]));
  });
  movieRecords.forEach((movie) => {
    const normalizedTitle = String(movie.searchTitle || movie.title).toLowerCase().replace(/[^a-z0-9]+/g, "");
    const year = Number(movie.localYear || movie.year || 0);
    const keys = [
      normalizedTitle ? `title:${normalizedTitle}` : "",
      movie.tmdbID > 0 ? `tmdb:${movie.tmdbID}` : "",
    ].filter(Boolean);
    const related = new Set(keys.flatMap((key) => identityGroups.get(key) || []).filter((candidate) => {
      const candidateYear = Number(candidate.localYear || candidate.year || 0);
      return (movie.tmdbID > 0 && candidate.tmdbID === movie.tmdbID) || year === 0 || candidateYear === 0 || year === candidateYear;
    }));
    related.delete(movie);
    movie.duplicate = related.size > 0;
    movie.renameCandidate = movie.missing ? ([...related].find((candidate) => !candidate.missing) || null) : null;
  });
  return movieRecords;
}

function tvShowFromRecord(show, index) {
  show = show || {};
  index = Math.max(0, Number(index) || 0);
  const title = show.name || show.local_title || "Unknown TV Show";
  const posterURL = artworkURL(show.poster_local_path);
  const fallbackPoster = [
    "linear-gradient(145deg, #151923 0%, #582626 50%, #07090d 100%)",
    "linear-gradient(145deg, #121f19 0%, #315b45 50%, #070b09 100%)",
    "linear-gradient(145deg, #111725 0%, #244c69 50%, #05080d 100%)",
    "linear-gradient(145deg, #201521 0%, #6f3d89 50%, #09050d 100%)",
  ][index % 4];

  return {
    id: show.id,
    title,
    searchTitle: show.local_title || title,
    year: show.year || show.local_year || "",
    seasons: show.seasons || "Unknown",
    episodes: show.episodes || show.local_episode_count || "Unknown",
    episodePaths: Array.isArray(show.episode_paths) ? show.episode_paths.filter(Boolean) : [],
    episodeMetadata: Array.isArray(show.episode_metadata) ? show.episode_metadata.filter(Boolean).map(episodeFromRecord) : [],
    path: show.path || "",
    tmdbID: Number(show.tmdb_id || 0),
    localEpisodeCount: Number(show.local_episode_count || 0),
    lastSeenAt: show.last_seen_at || "",
    missingSince: show.missing_since || "",
    missing: Boolean(show.missing_since),
    genres: Array.isArray(show.genres) ? show.genres.filter(Boolean) : [],
    network: show.network || "",
    studio: show.studio || "",
    actors: Array.isArray(show.actors) ? show.actors.filter(Boolean) : [],
    directors: Array.isArray(show.directors) ? show.directors.filter(Boolean) : [],
    writers: Array.isArray(show.writers) ? show.writers.filter(Boolean) : [],
    countries: Array.isArray(show.countries) ? show.countries.filter(Boolean) : [],
    languages: Array.isArray(show.languages) ? show.languages.filter(Boolean) : [],
    seriesStatus: show.series_status || "",
    firstAirDate: show.first_air_date || "",
    rating: show.rating || "",
    ratingValue: show.rating || "",
    tags: Array.isArray(show.tags) ? show.tags.filter(Boolean) : [],
    added: show.created_at,
    overview: show.overview || "Metadata is not available for this show.",
    poster: posterURL ? `url("${posterURL}") center / cover` : fallbackPoster,
  };
}

function reconcileTVShowStatuses(shows) {
  const titleGroups = new Map();
  const tmdbGroups = new Map();
  shows.forEach((show) => {
    const titleKey = String(show.searchTitle || show.title).toLowerCase().replace(/[^a-z0-9]+/g, "");
    if (titleKey) {
      titleGroups.set(titleKey, [...(titleGroups.get(titleKey) || []), show]);
    }
    if (show.tmdbID > 0) {
      tmdbGroups.set(show.tmdbID, [...(tmdbGroups.get(show.tmdbID) || []), show]);
    }
  });
  shows.forEach((show) => {
    const titleKey = String(show.searchTitle || show.title).toLowerCase().replace(/[^a-z0-9]+/g, "");
    const related = new Set([...(titleGroups.get(titleKey) || []), ...(tmdbGroups.get(show.tmdbID) || [])]);
    related.delete(show);
    show.duplicate = related.size > 0;
    if (show.missing) {
      show.renameCandidate = [...related].find((candidate) => !candidate.missing && (
        (show.tmdbID > 0 && candidate.tmdbID === show.tmdbID) ||
        (show.localEpisodeCount > 0 && candidate.localEpisodeCount === show.localEpisodeCount)
      )) || null;
    }
  });
  return shows;
}

function episodeFromRecord(episode) {
  return {
    ...episode,
    mediaMetadata: episode.media_metadata || {},
  };
}

function artworkURL(path) {
  return localAssetURL(path, "data/artwork/", "/artwork/");
}

function imageURL(path) {
  if (!path) {
    return "";
  }

  return artworkURL(path);
}

function localAssetURL(path, localPrefix, publicPrefix) {
  if (!path) {
    return "";
  }

  const normalized = path.replaceAll("\\", "/");
  const relativeIndex = normalized.indexOf(localPrefix);
  if (relativeIndex >= 0) {
    return `${publicPrefix}${normalized.slice(relativeIndex + localPrefix.length)}`;
  }

  return normalized.startsWith("/") ? normalized : "";
}

async function loadCollections() {
  if (!collectionGrid) return;
  try {
    const response = await fetch("/api/collections");
    const payload = await readJSONResponse(response);
    if (!response.ok) throw new Error(payload.error || "Could not load collections");
    collections = payload.collections || [];
    renderCollections();
    renderProgrammingPage();
  } catch (error) {
    if (collectionSaveStatus) collectionSaveStatus.textContent = error.message;
  }
}

function renderCollections() {
  if (!collectionGrid) return;
  if (collectionCount) collectionCount.textContent = collections.length.toLocaleString();
  collectionsEmptyState.hidden = collections.length > 0;
  collectionGrid.hidden = collections.length === 0;
  if (selectedCollection) {
    selectedCollection = collections.find((collection) => collection.id === selectedCollection.id) || null;
  }
  if (!selectedCollection && collections.length) selectedCollection = collections[0];
  const cards = collections.map((collection) => {
    const card = document.createElement("button");
    card.type = "button";
    card.className = `collection-card${collection.id === selectedCollection?.id ? " selected" : ""}`;
    card.innerHTML = `
      <span class="collection-card-art"><i></i><i></i><strong class="collection-auto-art-title" aria-hidden="true"></strong></span>
      <strong>${escapeHTML(collection.name)}</strong>
      <span class="collection-card-count">Calculating items…</span>`;
    card.addEventListener("click", () => {
      selectedCollection = collection;
      renderCollections();
    });
    previewCollection(collection).then((preview) => {
      const media = mergeCollectionMedia(preview.items, collection.pinned_media || []);
      applySavedCollectionArtwork(card.querySelector(".collection-card-art"), collection, media);
      const count = card.querySelector(".collection-card-count");
      if (count) count.textContent = `${preview.count.toLocaleString()} ${preview.count === 1 ? "item" : "items"}`;
    }).catch(() => {});
    return card;
  });
  collectionGrid.replaceChildren(...cards);
  renderCollectionInspector(selectedCollection);
}

async function renderCollectionInspector(collection) {
  if (!collectionInspector) return;
  if (!collection) {
    collectionInspector.innerHTML = `<section class="collection-inspector-empty"><h2>No collection selected</h2><p>Create a collection to inspect its membership and rules.</p></section>`;
    return;
  }
  collectionInspector.innerHTML = `<section class="collection-inspector-empty"><p>Loading collection details…</p></section>`;
  try {
    const preview = await previewCollection(collection);
    if (selectedCollection?.id !== collection.id) return;
    const rules = (collection.rules || []).map((rule) => `<span>${escapeHTML(rule.field)} ${escapeHTML(rule.operator.replaceAll("_", " "))} ${escapeHTML(rule.value)}</span>`).join("");
    collectionInspector.innerHTML = `
      <div class="collection-inspector-art"><i></i><i></i><strong class="collection-auto-art-title" aria-hidden="true"></strong></div>
      <h2>${escapeHTML(collection.name)}</h2>
      <p class="collection-inspector-count">${preview.count.toLocaleString()} ${preview.count === 1 ? "item" : "items"}</p>
      <p class="collection-inspector-description">${escapeHTML(collection.description || "No description provided.")}</p>
      <dl>
        <div><dt>Created</dt><dd>${escapeHTML(formatDate(collection.created_at))}</dd></div>
        <div><dt>Updated</dt><dd>${escapeHTML(formatDate(collection.updated_at))}</dd></div>
        <div><dt>Rules</dt><dd>${(collection.rules || []).length}</dd></div>
        <div><dt>Pinned</dt><dd>${(collection.pinned_media || []).length}</dd></div>
      </dl>
      ${rules ? `<section class="collection-inspector-rules"><h3>Rules</h3><div>${rules}</div></section>` : ""}
      <div class="collection-inspector-actions">
        <button class="button primary edit-collection-button" type="button">Edit Collection</button>
        <button class="button danger delete-collection-button" type="button">Delete Collection</button>
      </div>`;
    applySavedCollectionArtwork(collectionInspector.querySelector(".collection-inspector-art"), collection, mergeCollectionMedia(preview.items, collection.pinned_media || []));
    collectionInspector.querySelector(".edit-collection-button")?.addEventListener("click", () => openCollectionEditor(collection));
    collectionInspector.querySelector(".delete-collection-button")?.addEventListener("click", () => deleteCollection(collection));
  } catch (error) {
    collectionInspector.innerHTML = `<section class="collection-inspector-empty"><p>${escapeHTML(error.message)}</p></section>`;
  }
}

async function deleteCollection(collection) {
  if (!window.confirm(`Delete collection "${collection.name}"?`)) return;
  const response = await fetch(`/api/collections/${collection.id}`, { method: "DELETE" });
  if (!response.ok) {
    const payload = await readJSONResponse(response);
    window.alert(payload.error || "Could not delete collection");
    return;
  }
  collectionPreviewCache.delete(collection.id);
  selectedCollection = null;
  await loadCollections();
}

function openCollectionEditor(collection = null) {
  editingCollection = collection;
  collectionsHeader.hidden = true;
  collectionsWorkspace.hidden = true;
  collectionEditor.hidden = false;
  collectionEditorTitle.textContent = collection ? "Edit Collection" : "Create Collection";
  collectionNameInput.value = collection?.name || "";
  collectionDescriptionInput.value = collection?.description || "";
  collectionArtworkPath = collection?.artwork_path || "";
  collectionArtworkMode = collection?.artwork_mode || (collectionArtworkPath ? "custom" : "auto");
  collectionArtworkMedia = [...(collection?.artwork_media || [])];
  collectionSaveStatus.textContent = "";
  collectionPinnedMedia = [...(collection?.pinned_media || [])];
  collectionExcludedMedia = [...(collection?.excluded_media || [])];
  collectionPreviewTotal = 0;
  if (collectionPreviewSearch) collectionPreviewSearch.value = "";
  if (collectionPreviewShowExcluded) collectionPreviewShowExcluded.checked = false;
  collectionRules.replaceChildren(...(collection?.rules || []).map(createCollectionRuleRow));
  collectionPreviewItems = [];
  renderPinnedCollectionMedia();
  renderCollectionMediaResults();
  if (collection?.id && collectionPreviewCache.has(collection.id)) {
    applyCollectionPreview(collectionPreviewCache.get(collection.id), collectCollectionRules());
  } else if (collection?.id) {
    collectionMatchSummary.textContent = "Loading matching media…";
    previewCollection(collection).then((preview) => {
      if (editingCollection?.id === collection.id) applyCollectionPreview(preview, collectCollectionRules());
    }).catch((error) => {
      if (editingCollection?.id === collection.id) collectionMatchSummary.textContent = error.message;
    });
  } else {
    collectionMatchSummary.textContent = "No rules configured. Add rules or pin media below.";
    scheduleCollectionPreview();
  }
  collectionNameInput.focus();
}

function closeCollectionEditor() {
  editingCollection = null;
  collectionEditor.hidden = true;
  collectionsHeader.hidden = false;
  collectionsWorkspace.hidden = false;
  collectionSaveStatus.textContent = "";
  renderCollections();
}

function createCollectionRuleRow(rule = {}) {
  const row = document.createElement("div");
  row.className = "collection-rule-row";
  const mediaType = rule.media_type === "tv" ? "tv" : "movie";
  const fieldLabel = availableRuleLabels(mediaType).find((label) => ruleOptions[label]?.field === rule.field) || availableRuleLabels(mediaType)[0];
  row.innerHTML = `
    <span class="collection-rule-handle" aria-hidden="true">⠿</span>
    <select class="collection-rule-media" aria-label="Media type"></select>
    <select class="collection-rule-field" aria-label="Metadata field"></select>
    <select class="collection-rule-operator" aria-label="Rule operator"></select>
    <select class="collection-rule-value" aria-label="Rule value"></select>
    <button class="collection-rule-delete" type="button" aria-label="Delete rule">×</button>`;
  const mediaSelect = row.querySelector(".collection-rule-media");
  const fieldSelect = row.querySelector(".collection-rule-field");
  populateSelect(mediaSelect, ruleMediaTypes, mediaType);
  populateSelect(fieldSelect, availableRuleLabels(mediaType), fieldLabel);
  updateCollectionRuleRow(row, rule.operator, rule.value);
  mediaSelect.addEventListener("change", () => {
    populateSelect(fieldSelect, availableRuleLabels(mediaSelect.value), "");
    updateCollectionRuleRow(row);
    scheduleCollectionPreview();
  });
  fieldSelect.addEventListener("change", () => {
    updateCollectionRuleRow(row);
    scheduleCollectionPreview();
  });
  row.querySelector(".collection-rule-operator").addEventListener("change", scheduleCollectionPreview);
  row.querySelector(".collection-rule-value").addEventListener("change", scheduleCollectionPreview);
  row.querySelector(".collection-rule-delete").addEventListener("click", () => {
    row.remove();
    scheduleCollectionPreview();
  });
  return row;
}

function updateCollectionRuleRow(row, operator = "", value = "") {
  const mediaType = row.querySelector(".collection-rule-media").value || "movie";
  const field = row.querySelector(".collection-rule-field").value;
  const options = ruleOptions[field];
  if (!options) return;
  populateSelect(row.querySelector(".collection-rule-operator"), options.operators, operator);
  populateSelect(row.querySelector(".collection-rule-value"), ruleValuesFor(mediaType, field), value);
}

function collectCollectionRules() {
  return [...collectionRules.querySelectorAll(".collection-rule-row")].map((row) => {
    const mediaType = row.querySelector(".collection-rule-media").value || "movie";
    const fieldLabel = row.querySelector(".collection-rule-field").value;
    return {
      media_type: mediaType,
      field: ruleOptions[fieldLabel]?.field || fieldLabel.toLowerCase(),
      operator: row.querySelector(".collection-rule-operator").value,
      value: row.querySelector(".collection-rule-value").value,
    };
  }).filter((rule) => rule.field && rule.operator && rule.value);
}

function scheduleCollectionPreview() {
  window.clearTimeout(collectionPreviewTimer);
  collectionPreviewTimer = window.setTimeout(refreshCollectionPreview, 180);
}

async function previewCollection(collection) {
  if (collection.id && collectionPreviewCache.has(collection.id)) return collectionPreviewCache.get(collection.id);
  if (collection.id && collectionPreviewRequests.has(collection.id)) return collectionPreviewRequests.get(collection.id);
  const request = (async () => {
    const response = await fetch("/api/collections/preview", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ rules: collection.rules || [], pinned_media: collection.pinned_media || [], excluded_media: collection.excluded_media || [] }),
    });
    const payload = await readJSONResponse(response);
    if (!response.ok) throw new Error(payload.error || "Could not preview collection");
    const preview = { items: payload.items || [], count: Number(payload.count || 0) };
    if (collection.id) collectionPreviewCache.set(collection.id, preview);
    return preview;
  })();
  if (collection.id) collectionPreviewRequests.set(collection.id, request);
  try {
    return await request;
  } finally {
    if (collection.id) collectionPreviewRequests.delete(collection.id);
  }
}

async function refreshCollectionPreview() {
  const rules = collectCollectionRules();
  try {
    const preview = await previewCollection({ rules, pinned_media: collectionPinnedMedia, excluded_media: collectionExcludedMedia });
    applyCollectionPreview(preview, rules);
  } catch (error) {
    collectionMatchSummary.textContent = error.message;
    if (collectionPreviewList) collectionPreviewList.innerHTML = `<p class="collection-preview-empty">${escapeHTML(error.message)}</p>`;
  }
}

function applyCollectionPreview(preview, rules) {
  collectionPreviewItems = preview.items || [];
  collectionPreviewTotal = Number(preview.count || 0);
  collectionMatchSummary.textContent = rules.length
    ? `${collectionPreviewTotal.toLocaleString()} media items currently match this collection.`
    : `${collectionPreviewTotal.toLocaleString()} pinned items included. Add rules to include matching media automatically.`;
  updateCollectionArtwork();
  renderCollectionPreviewPanel();
}

function allLocalCollectionMedia() {
  return [
    ...movies.map((movie) => ({ media_type: "movie", media_id: movie.id, title: movie.title, poster: movie.poster })),
    ...tvShows.map((show) => ({ media_type: "tv", media_id: show.id, title: show.title, poster: show.poster })),
  ];
}

function mediaKey(media) {
  return `${media.media_type}:${media.media_id}`;
}

function mergeCollectionMedia(previewItems, pinned) {
  const localByKey = new Map(allLocalCollectionMedia().map((item) => [mediaKey(item), item]));
  const normalizedPreview = previewItems.map((item) => ({
    ...item,
    poster: item.poster || (item.poster_url ? `url("${item.poster_url}") center / cover` : ""),
  }));
  const combined = [...pinned.map((item) => localByKey.get(mediaKey(item))).filter(Boolean), ...normalizedPreview];
  return [...new Map(combined.map((item) => [mediaKey(item), item])).values()];
}

function renderCollectionPreviewPanel() {
  if (!collectionPreviewList) return;
  const excludedKeys = new Set(collectionExcludedMedia.map(mediaKey));
  const localByKey = new Map(allLocalCollectionMedia().map((item) => [mediaKey(item), item]));
  const included = mergeCollectionMedia(collectionPreviewItems, collectionPinnedMedia).filter((item) => !excludedKeys.has(mediaKey(item)));
  const excluded = collectionExcludedMedia.map((identity) => localByKey.get(mediaKey(identity))).filter(Boolean);
  const query = String(collectionPreviewSearch?.value || "").trim().toLowerCase();
  const showingExcluded = Boolean(collectionPreviewShowExcluded?.checked);
  const visible = (showingExcluded ? [...included, ...excluded] : included)
    .filter((item) => !query || item.title.toLowerCase().includes(query))
    .sort((left, right) => left.title.localeCompare(right.title));
  if (collectionPreviewCount) collectionPreviewCount.textContent = `${collectionPreviewTotal.toLocaleString()} included · ${collectionExcludedMedia.length.toLocaleString()} excluded`;
  if (!visible.length) {
    collectionPreviewList.innerHTML = `<p class="collection-preview-empty">${query ? "No matching titles found." : showingExcluded ? "No included or excluded media yet." : "Add or adjust rules to see matching media."}</p>`;
    return;
  }
  const rows = visible.map((item) => {
    const isExcluded = excludedKeys.has(mediaKey(item));
    const row = document.createElement("article");
    row.className = `collection-preview-item${isExcluded ? " excluded" : ""}`;
    row.innerHTML = `<span class="collection-preview-item-art" style="--poster:${escapeAttribute(item.poster || "")}"></span><span><strong>${escapeHTML(item.title)}</strong><small>${item.media_type === "tv" ? "TV Show" : "Movie"}</small></span><button type="button">${isExcluded ? "Restore" : "Exclude"}</button>`;
    row.querySelector("button").addEventListener("click", () => {
      if (isExcluded) collectionExcludedMedia = collectionExcludedMedia.filter((media) => mediaKey(media) !== mediaKey(item));
      else if (!excludedKeys.has(mediaKey(item))) collectionExcludedMedia.push({ media_type: item.media_type, media_id: item.media_id });
      renderCollectionPreviewPanel();
      refreshCollectionPreview();
    });
    return row;
  });
  collectionPreviewList.replaceChildren(...rows);
}

function renderCollectionMediaResults() {
  const query = collectionMediaSearch.value.trim().toLowerCase();
  if (!query) {
    collectionMediaResults.replaceChildren();
    return;
  }
  const pinnedKeys = new Set(collectionPinnedMedia.map(mediaKey));
  const results = allLocalCollectionMedia()
    .filter((item) => !pinnedKeys.has(mediaKey(item)) && item.title.toLowerCase().includes(query))
    .slice(0, 8)
    .map((item) => {
      const button = document.createElement("button");
      button.type = "button";
      button.className = "collection-media-result";
      button.innerHTML = `<span style="--poster:${escapeAttribute(item.poster)}"></span><strong>${escapeHTML(item.title)}</strong><small>${item.media_type === "tv" ? "TV Show" : "Movie"}</small>`;
      button.addEventListener("click", () => {
        collectionPinnedMedia.push({ media_type: item.media_type, media_id: item.media_id });
        collectionMediaSearch.value = "";
        renderCollectionMediaResults();
        renderPinnedCollectionMedia();
        refreshCollectionPreview();
      });
      return button;
    });
  collectionMediaResults.replaceChildren(...results);
}

function renderPinnedCollectionMedia() {
  const localByKey = new Map(allLocalCollectionMedia().map((item) => [mediaKey(item), item]));
  const chips = collectionPinnedMedia.map((pinned) => {
    const item = localByKey.get(mediaKey(pinned));
    if (!item) return null;
    const chip = document.createElement("article");
    chip.className = "collection-pinned-item";
    chip.innerHTML = `<span style="--poster:${escapeAttribute(item.poster)}"></span><div><strong>${escapeHTML(item.title)}</strong><small>${item.media_type === "tv" ? "TV Show" : "Movie"}</small></div><button type="button" aria-label="Remove ${escapeAttribute(item.title)}">×</button>`;
    chip.querySelector("button").addEventListener("click", () => {
      collectionPinnedMedia = collectionPinnedMedia.filter((media) => mediaKey(media) !== mediaKey(pinned));
      renderPinnedCollectionMedia();
      refreshCollectionPreview();
    });
    return chip;
  }).filter(Boolean);
  collectionPinnedMediaContainer.replaceChildren(...chips);
  updateCollectionArtwork();
}

function updateCollectionArtwork() {
  if (collectionArtworkMode === "custom" && collectionArtworkPath) {
    collectionArtworkPreview.classList.add("custom-artwork", "has-artwork");
    collectionArtworkPreview.style.backgroundImage = `url("${collectionArtworkPath.replaceAll('"', '%22')}")`;
    if (shuffleCollectionArtwork) shuffleCollectionArtwork.hidden = true;
    if (removeCollectionArtwork) removeCollectionArtwork.hidden = false;
    return;
  }
  collectionArtworkPreview.classList.remove("custom-artwork");
  collectionArtworkPreview.style.backgroundImage = "";
  if (shuffleCollectionArtwork) shuffleCollectionArtwork.hidden = false;
  if (removeCollectionArtwork) removeCollectionArtwork.hidden = true;
  const excludedKeys = new Set(collectionExcludedMedia.map(mediaKey));
  const candidates = mergeCollectionMedia(collectionPreviewItems, collectionPinnedMedia).filter((item) => !excludedKeys.has(mediaKey(item)));
  const candidatesByKey = new Map(candidates.map((item) => [mediaKey(item), item]));
  let selected = collectionArtworkMedia.map((identity) => candidatesByKey.get(mediaKey(identity))).filter(Boolean);
  if (selected.length !== 2) {
    selected = chooseCollectionArtworkMedia(candidates, collectionNameInput.value);
    collectionArtworkMedia = selected.map((item) => ({ media_type: item.media_type, media_id: item.media_id }));
  }
  applyTwoPosterArtwork(
    collectionArtworkPreview,
    selected,
    collectionNameInput.value,
  );
}

function chooseCollectionArtworkMedia(media, seedText = "", randomize = false) {
  const usable = [...new Map(media.filter((item) => item.poster).map((item) => [mediaKey(item), item])).values()];
  if (usable.length <= 2) return usable;
  let seed = randomize
    ? Math.floor(Math.random() * 0x7fffffff)
    : [...String(seedText)].reduce((total, char) => total * 31 + char.charCodeAt(0), usable.length * 17);
  const selected = [];
  while (usable.length && selected.length < 2) {
    const index = Math.abs(seed) % usable.length;
    selected.push(usable.splice(index, 1)[0]);
    seed = seed * 31 + 7;
  }
  return selected;
}

function shuffleAutoCollectionArtwork() {
  const excludedKeys = new Set(collectionExcludedMedia.map(mediaKey));
  const candidates = mergeCollectionMedia(collectionPreviewItems, collectionPinnedMedia).filter((item) => !excludedKeys.has(mediaKey(item)));
  const previous = collectionArtworkMedia.map(mediaKey).join("|");
  let selected = chooseCollectionArtworkMedia(candidates, collectionNameInput.value, true);
  for (let attempt = 0; attempt < 5 && candidates.length > 2 && selected.map(mediaKey).join("|") === previous; attempt += 1) {
    selected = chooseCollectionArtworkMedia(candidates, collectionNameInput.value, true);
  }
  collectionArtworkMedia = selected.map((item) => ({ media_type: item.media_type, media_id: item.media_id }));
  updateCollectionArtwork();
}

async function uploadCollectionArtwork() {
  const file = collectionArtworkFile?.files?.[0];
  if (!file) return;
  const form = new FormData();
  form.append("artwork", file);
  chooseCollectionArtwork.disabled = true;
  collectionSaveStatus.textContent = "Uploading artwork…";
  try {
    const response = await fetch("/api/collection-artwork", { method: "POST", body: form });
    const payload = await readJSONResponse(response);
    if (!response.ok) throw new Error(payload.error || "Could not upload artwork");
    collectionArtworkPath = payload.artwork_path;
    collectionArtworkMode = "custom";
    collectionArtworkMedia = [];
    collectionSaveStatus.textContent = "Artwork uploaded. Save the collection to keep it.";
    updateCollectionArtwork();
  } catch (error) {
    collectionSaveStatus.textContent = error.message;
  } finally {
    chooseCollectionArtwork.disabled = false;
    collectionArtworkFile.value = "";
  }
}

function applyTwoPosterArtwork(container, media, seedText = "") {
  if (!container) return;
  const halves = container.querySelectorAll("[data-artwork-half], i");
  const usable = media.filter((item) => item.poster);
  const selected = usable.slice(0, 2);
  halves.forEach((half, index) => {
    half.style.display = "";
    half.style.background = selected[index]?.poster || "linear-gradient(145deg, #111827, #251332)";
    half.style.backgroundPosition = "center top";
    half.style.backgroundRepeat = "no-repeat";
    half.style.backgroundSize = "contain";
  });
  const title = container.querySelector(".collection-auto-art-title");
  if (title) title.textContent = String(seedText || "Untitled Collection").trim();
  container.classList.toggle("has-artwork", selected.length >= 2);
}

function applySavedCollectionArtwork(container, collection, media) {
  if (collection?.artwork_path) {
    container.style.background = `#02050a url("${collection.artwork_path.replaceAll('"', '%22')}") center / contain no-repeat`;
    container.classList.add("has-artwork", "custom-artwork");
    container.querySelectorAll("i").forEach((item) => { item.style.display = "none"; });
    return;
  }
  applyTwoPosterArtwork(container, media, collection?.name || "");
}

async function saveCollection() {
  const name = collectionNameInput.value.trim();
  if (!name) {
    collectionSaveStatus.textContent = "Collection name is required.";
    collectionNameInput.focus();
    return;
  }
  saveCollectionButton.disabled = true;
  collectionSaveStatus.textContent = "Saving collection…";
  const payload = {
    name,
    description: collectionDescriptionInput.value.trim(),
    artwork_path: collectionArtworkPath,
    artwork_mode: collectionArtworkMode,
    artwork_media: collectionArtworkMode === "auto" ? collectionArtworkMedia : [],
    rules: collectCollectionRules(),
    pinned_media: collectionPinnedMedia,
    excluded_media: collectionExcludedMedia,
  };
  try {
    const response = await fetch(editingCollection?.id ? `/api/collections/${editingCollection.id}` : "/api/collections", {
      method: editingCollection?.id ? "PUT" : "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
    const saved = await readJSONResponse(response);
    if (!response.ok) throw new Error(saved.error || "Could not save collection");
    collectionPreviewCache.delete(saved.id);
    selectedCollection = saved;
    await loadCollections();
    closeCollectionEditor();
  } catch (error) {
    collectionSaveStatus.textContent = error.message;
  } finally {
    saveCollectionButton.disabled = false;
  }
}

async function loadGuide(dateKey = guideDateValue, scrollToNow = true) {
  if (!guideStatus) return;
  const cached = guideDayCache.get(dateKey);
  if (cached) {
    guideDateValue = dateKey;
    guideChannels = cached.channels;
    guidePrograms = cached.programs;
    selectedGuideProgram = null;
    if (guideDate) guideDate.value = dateKey;
    guideLoaded = true;
    renderGuide(scrollToNow);
    return;
  }

  const requestSequence = ++guideLoadSequence;
  guideStatus.textContent = "Loading guide…";
  try {
    const [year, month, day] = dateKey.split("-").map(Number);
    const windowStart = new Date(year, month - 1, day, 0, 0, 0, 0);
    const windowEnd = new Date(year, month - 1, day + 1, 0, 0, 0, 0);
    const windowHours = Math.round((windowEnd - windowStart) / 3600000);
    const params = new URLSearchParams({ start: windowStart.toISOString(), hours: String(windowHours) });
    const response = await fetch(`/guide.xml?${params}`);
    if (!response.ok) throw new Error(`Could not load guide (${response.status})`);
    const documentXML = new DOMParser().parseFromString(await response.text(), "application/xml");
    if (documentXML.querySelector("parsererror")) throw new Error("The guide endpoint returned invalid XML");

    const channelsForDay = [...documentXML.querySelectorAll("tv > channel")].map((channel, index) => ({
      id: channel.getAttribute("id") || `channel-${index + 1}`,
      number: index + 1,
      name: channel.querySelector("display-name")?.textContent?.trim() || `Channel ${index + 1}`,
      logo: channel.querySelector("icon")?.getAttribute("src") || "",
    }));
    const programsForDay = [...documentXML.querySelectorAll("tv > programme")].map((program, index) => ({
      id: `guide-program-${index}`,
      channelID: program.getAttribute("channel") || "",
      start: parseXMLTVTime(program.getAttribute("start")),
      stop: parseXMLTVTime(program.getAttribute("stop")),
      title: program.querySelector("title")?.textContent?.trim() || "Untitled program",
      description: program.querySelector("desc")?.textContent?.trim() || "No program description is available.",
      artwork: program.querySelector("icon")?.getAttribute("src") || "",
    })).filter((program) => program.start && program.stop);
    if (requestSequence !== guideLoadSequence) return;
    guideDayCache.set(dateKey, { channels: channelsForDay, programs: programsForDay });
    guideDateValue = dateKey;
    guideChannels = channelsForDay;
    guidePrograms = programsForDay;
    selectedGuideProgram = null;
    guideLoaded = true;
    if (guideDate) {
      guideDate.value = dateKey;
      guideDate.min = localDateKey(new Date());
      const lastGuideDay = new Date();
      lastGuideDay.setDate(lastGuideDay.getDate() + 7);
      guideDate.max = localDateKey(lastGuideDay);
    }
    guideStatus.textContent = `${guideChannels.length.toLocaleString()} airing channels`;
    renderGuide(scrollToNow);
  } catch (error) {
    if (requestSequence !== guideLoadSequence) return;
    guideStatus.textContent = error.message;
    guideSchedule?.replaceChildren();
    renderGuideInspector(null);
  }
}

function parseXMLTVTime(value = "") {
  const match = value.match(/^(\d{4})(\d{2})(\d{2})(\d{2})(\d{2})(\d{2})/);
  if (!match) return null;
  return new Date(Date.UTC(
    Number(match[1]), Number(match[2]) - 1, Number(match[3]),
    Number(match[4]), Number(match[5]), Number(match[6]),
  ));
}

function localDateKey(value) {
  const date = value instanceof Date ? value : new Date(value);
  const pad = (part) => String(part).padStart(2, "0");
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
}

function guideDayStart() {
  const [year, month, day] = guideDateValue.split("-").map(Number);
  return new Date(year, month - 1, day, 0, 0, 0, 0);
}

function renderGuide(scrollToNow = false) {
  if (!guideSchedule || !guideTimeHeader) return;
  const dayStart = guideDayStart();
  const dayEnd = new Date(dayStart.getTime() + 24 * 60 * 60 * 1000);
  const trackWidth = 24 * guidePixelsPerHour;
  const visiblePrograms = guidePrograms.filter((program) => program.start < dayEnd && program.stop > dayStart);
  if (guideStatus) {
    const dateLabel = new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(dayStart);
    guideStatus.textContent = `${guideChannels.length.toLocaleString()} airing channels · ${visiblePrograms.length.toLocaleString()} actual programs on ${dateLabel}`;
  }

  const spacer = document.createElement("div");
  spacer.className = "guide-time-spacer";
  const timeline = document.createElement("div");
  timeline.className = "guide-time-scale";
  timeline.style.width = `${trackWidth}px`;
  for (let hour = 0; hour < 24; hour++) {
    const tick = document.createElement("span");
    tick.style.left = `${hour * guidePixelsPerHour}px`;
    tick.textContent = new Intl.DateTimeFormat(undefined, { hour: "numeric" }).format(new Date(dayStart.getTime() + hour * 60 * 60 * 1000));
    timeline.append(tick);
  }
  guideTimeHeader.replaceChildren(spacer, timeline);

  const now = new Date();
  const showNow = localDateKey(now) === guideDateValue;
  if (showNow) {
    const nowOffset = ((now - dayStart) / 3600000) * guidePixelsPerHour;
    const headerMarker = document.createElement("span");
    headerMarker.className = "guide-now-header";
    headerMarker.style.left = `${nowOffset}px`;
    headerMarker.textContent = formatGuideTime(now);
    timeline.append(headerMarker);
  }
  if (!visiblePrograms.includes(selectedGuideProgram)) {
    selectedGuideProgram = visiblePrograms.find((program) => program.start <= now && program.stop > now) || visiblePrograms[0] || null;
  }
  const rows = guideChannels.map((channel) => {
    const row = document.createElement("section");
    row.className = "guide-channel-row";
    const identity = document.createElement("div");
    identity.className = "guide-channel-identity";
    const number = document.createElement("span");
    number.className = "guide-channel-number";
    number.textContent = String(channel.number);
    const logo = document.createElement("span");
    logo.className = "guide-channel-logo";
    if (channel.logo) logo.style.backgroundImage = `url("${channel.logo.replaceAll('"', '%22')}")`;
    const name = document.createElement("strong");
    name.textContent = channel.name;
    identity.append(number, logo, name);

    const track = document.createElement("div");
    track.className = "guide-channel-track";
    track.style.width = `${trackWidth}px`;
    visiblePrograms.filter((program) => program.channelID === channel.id).forEach((program) => {
      const clippedStart = Math.max(program.start.getTime(), dayStart.getTime());
      const clippedStop = Math.min(program.stop.getTime(), dayEnd.getTime());
      const left = ((clippedStart - dayStart.getTime()) / 3600000) * guidePixelsPerHour;
      const width = Math.max(44, ((clippedStop - clippedStart) / 3600000) * guidePixelsPerHour - 4);
      const button = document.createElement("button");
      button.type = "button";
      button.className = `guide-program${program === selectedGuideProgram ? " selected" : ""}`;
      button.style.left = `${left}px`;
      button.style.width = `${width}px`;
      button.innerHTML = `<strong>${escapeHTML(program.title)}</strong><span>${escapeHTML(formatGuideTime(program.start))} – ${escapeHTML(formatGuideTime(program.stop))}</span>`;
      button.addEventListener("click", () => {
        selectedGuideProgram = program;
        renderGuideInspector(program);
        guideSchedule.querySelectorAll(".guide-program").forEach((item) => item.classList.toggle("selected", item === button));
      });
      track.append(button);
    });
    if (showNow) {
      const marker = document.createElement("span");
      marker.className = "guide-now-line";
      marker.style.left = `${((now - dayStart) / 3600000) * guidePixelsPerHour}px`;
      track.append(marker);
    }
    row.append(identity, track);
    return row;
  });
  guideSchedule.replaceChildren(...rows);

  renderGuideInspector(selectedGuideProgram);
  if (scrollToNow && showNow) {
    requestAnimationFrame(() => {
      const shell = document.querySelector(".guide-schedule-shell");
      if (shell) shell.scrollLeft = Math.max(0, ((now - dayStart) / 3600000) * guidePixelsPerHour - 300);
    });
  }
}

function renderGuideInspector(program) {
  if (!guideInspector) return;
  if (!program) {
    guideInspector.innerHTML = `<section class="guide-inspector-empty"><h2>No program selected</h2><p>Select a scheduled program to inspect its details.</p></section>`;
    return;
  }
  const channel = guideChannels.find((item) => item.id === program.channelID);
  const duration = Math.max(0, Math.round((program.stop - program.start) / 60000));
  guideInspector.innerHTML = `
    <div class="guide-inspector-art"></div>
    <h2>${escapeHTML(program.title)}</h2>
    <p class="guide-inspector-time">${escapeHTML(formatGuideTime(program.start))} – ${escapeHTML(formatGuideTime(program.stop))} · ${duration} min</p>
    <p class="guide-inspector-description">${escapeHTML(program.description)}</p>
    <dl>
      <div><dt>Channel</dt><dd>${escapeHTML(channel?.name || "Unknown")}</dd></div>
      <div><dt>Airs</dt><dd>${escapeHTML(new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(program.start))}</dd></div>
      <div><dt>Runtime</dt><dd>${duration} min</dd></div>
    </dl>`;
  const art = guideInspector.querySelector(".guide-inspector-art");
  if (program.artwork) art.style.backgroundImage = `url("${program.artwork.replaceAll('"', '%22')}")`;
}

function formatGuideTime(date) {
  return new Intl.DateTimeFormat(undefined, { hour: "numeric", minute: "2-digit" }).format(date);
}

function changeGuideDay(offset) {
  const date = guideDayStart();
  date.setDate(date.getDate() + offset);
  const nextDate = localDateKey(date);
  if ((guideDate?.min && nextDate < guideDate.min) || (guideDate?.max && nextDate > guideDate.max)) return;
  guideDateValue = nextDate;
  if (guideDate) guideDate.value = guideDateValue;
  loadGuide(guideDateValue, false);
}

search.addEventListener("input", () => {
  moviePage = 1;
  render();
});
[movieGenreFilter, movieYearFilter, movieRatingFilter, movieRuntimeFilter].forEach((control) => {
  control?.addEventListener("change", () => {
    moviePage = 1;
    render();
  });
});
[tvGenreFilter, tvYearFilter, tvRatingFilter, tvNetworkFilter].forEach((control) => {
  control?.addEventListener("change", () => {
    tvShowPage = 1;
    renderTVShows();
  });
});
movieSort?.addEventListener("change", () => {
  movieSortMode = movieSort.value;
  moviePage = 1;
  render();
});
clearMovieFiltersButton?.addEventListener("click", () => {
  const controls = currentView === "tv"
    ? [tvGenreFilter, tvYearFilter, tvRatingFilter, tvNetworkFilter]
    : [movieGenreFilter, movieYearFilter, movieRatingFilter, movieRuntimeFilter];
  controls.forEach((control) => {
    if (control) control.value = "";
  });
  if (currentView === "tv") {
    tvShowPage = 1;
    renderTVShows();
  } else {
    moviePage = 1;
    render();
  }
});
movieFilterToggle?.addEventListener("click", () => {
  document.querySelector(".library-sidebar")?.classList.toggle("filters-open");
});
tvFilterToggle?.addEventListener("click", () => {
  document.querySelector(".library-sidebar")?.classList.toggle("filters-open");
});
guideDate?.addEventListener("change", () => {
  if (!guideDate.value) return;
  guideDateValue = guideDate.value;
  loadGuide(guideDateValue, false);
});
guideTodayButton?.addEventListener("click", () => {
  guideDateValue = localDateKey(new Date());
  if (guideDate) guideDate.value = guideDateValue;
  loadGuide(guideDateValue, true);
});
guideNowButton?.addEventListener("click", () => {
  guideDateValue = localDateKey(new Date());
  if (guideDate) guideDate.value = guideDateValue;
  loadGuide(guideDateValue, true);
});
guidePreviousDayButton?.addEventListener("click", () => changeGuideDay(-1));
guideNextDayButton?.addEventListener("click", () => changeGuideDay(1));
createCollectionButton?.addEventListener("click", () => openCollectionEditor());
createFirstCollectionButton?.addEventListener("click", () => openCollectionEditor());
collectionBackButton?.addEventListener("click", closeCollectionEditor);
cancelCollectionButton?.addEventListener("click", closeCollectionEditor);
saveCollectionButton?.addEventListener("click", saveCollection);
addCollectionRuleButton?.addEventListener("click", () => {
  collectionRules.append(createCollectionRuleRow());
  scheduleCollectionPreview();
});
collectionMediaSearch?.addEventListener("input", renderCollectionMediaResults);
collectionPreviewSearch?.addEventListener("input", renderCollectionPreviewPanel);
collectionPreviewShowExcluded?.addEventListener("change", renderCollectionPreviewPanel);
collectionNameInput?.addEventListener("input", updateCollectionArtwork);
chooseCollectionArtwork?.addEventListener("click", () => collectionArtworkFile?.click());
collectionArtworkFile?.addEventListener("change", uploadCollectionArtwork);
removeCollectionArtwork?.addEventListener("click", () => {
  if (collectionArtworkMode === "custom" && collectionArtworkPath && !window.confirm("This will replace the currently uploaded artwork. Are you sure?")) return;
  collectionArtworkMode = "auto";
  collectionArtworkPath = "";
  collectionArtworkMedia = [];
  collectionSaveStatus.textContent = "Auto artwork selected. Save the collection to keep it.";
  updateCollectionArtwork();
});
shuffleCollectionArtwork?.addEventListener("click", shuffleAutoCollectionArtwork);
window.setInterval(() => {
  if (currentView === "guide" && guideLoaded && guideDateValue === localDateKey(new Date())) {
    renderGuide(false);
  }
}, 60 * 1000);
refreshSelectedMetadataButton?.addEventListener("click", () => {
  if (selectedMovie) refreshMovieMetadata(selectedMovie);
});
libraryFilterLinks.forEach((item) => {
  item.addEventListener("click", () => {
    movieLibraryMode = item.dataset.libraryMode || "all";
    libraryFilterLinks.forEach((link) => link.classList.toggle("active", link === item));
    document.querySelector('.nav-item[data-view="movies"]')?.classList.remove("active");
    setView("movies");
    item.classList.add("active");
    moviePage = 1;
    render();
  });
});
tvSearch?.addEventListener("input", () => {
  tvShowPage = 1;
  renderTVShows();
});
scanButton.addEventListener("click", scanLibrary);
scanTVButton?.addEventListener("click", scanTVShows);
saveSettingsButton?.addEventListener("click", saveSettings);
addSourceButton?.addEventListener("click", addSource);
addAssetSourceButton?.addEventListener("click", addAssetSource);
scanAssetsButton?.addEventListener("click", scanAssets);
streamingModeControl?.addEventListener("change", (event) => {
  if (event.target?.name === "streaming-mode") {
    streamingMode = event.target.value === "hls" ? "hls" : "direct";
    setSettingsDirty(true);
  }
});
scanAllSourcesButton?.addEventListener("click", scanAllSources);
channelSearch?.addEventListener("input", renderChannels);
createChannelButton?.addEventListener("click", newChannel);
exportChannelsButton?.addEventListener("click", exportChannels);
importChannelsButton?.addEventListener("click", () => importChannelsFileInput?.click());
importChannelsFileInput?.addEventListener("change", importChannels);
saveChannelButton?.addEventListener("click", saveChannel);
channelNameInput?.addEventListener("input", updateBuilderLogo);
channelLogoPathInput?.addEventListener("input", updateBuilderLogo);
channelContentTypeInput?.addEventListener("change", setupRuleBuilder);
channelLogoEditButton?.addEventListener("click", () => channelLogoFileInput?.click());
channelLogoFileInput?.addEventListener("change", uploadChannelLogo);
builderTabButtons.forEach((button) => {
  button.addEventListener("click", () => setBuilderTab(button.dataset.builderTab));
});
addRuleButton?.addEventListener("click", addRule);
addProgrammingBlockButton?.addEventListener("click", addProgrammingBlock);
addProgrammingEventButton?.addEventListener("click", addProgrammingEvent);
programmingBlockWeekdaysButton?.addEventListener("click", setProgrammingBlockWeekdays);
programmingBlockDays?.querySelectorAll("[data-block-day]").forEach((button) => {
  button.addEventListener("click", () => togglePendingProgrammingBlockDay(button.dataset.blockDay));
});
cancelProgrammingBlock?.addEventListener("click", closeProgrammingBlockModal);
cancelProgrammingBlockX?.addEventListener("click", closeProgrammingBlockModal);
createProgrammingBlockButton?.addEventListener("click", createProgrammingBlockFromModal);
programmingBlockModal?.addEventListener("click", (event) => {
  if (event.target === programmingBlockModal) {
    closeProgrammingBlockModal();
  }
});
programmingChannelSelect?.addEventListener("change", () => {
  programmingPageChannelID = programmingChannelSelect.value;
  const channel = selectedProgrammingChannel();
  programmingPageBlocks = dedupeProgrammingBlocks(normalizeProgrammingItems(channel?.programming_blocks, "block"));
  programmingPageSelectedBlockID = programmingPageBlocks[0]?.id || null;
  renderProgrammingPage();
});
programmingCollectionSearch?.addEventListener("input", renderProgrammingCollections);
programmingCollectionType?.addEventListener("change", renderProgrammingCollections);
programmingSourceTabs.forEach((tab) => tab.addEventListener("click", () => {
  programmingSourceTab = tab.dataset.programmingSource || "collections";
  programmingCollectionSearch.value = "";
  renderProgrammingCollections();
}));
document.querySelectorAll('input[name="programming-schedule-mode"]').forEach((input) => input.addEventListener("change", renderProgrammingDropMode));
programmingDropDays?.querySelectorAll("[data-drop-day]").forEach((button) => button.addEventListener("click", () => {
  const day = button.dataset.dropDay;
  pendingProgrammingDropDays = pendingProgrammingDropDays.includes(day) ? pendingProgrammingDropDays.filter((item) => item !== day) : [...pendingProgrammingDropDays, day];
  renderProgrammingDropMode();
}));
programmingDropClose?.addEventListener("click", closeProgrammingDropModal);
programmingDropCancel?.addEventListener("click", closeProgrammingDropModal);
programmingDropConfirm?.addEventListener("click", confirmProgrammingDrop);
programmingDropModal?.addEventListener("click", (event) => { if (event.target === programmingDropModal) closeProgrammingDropModal(); });
[programmingDropDate, programmingDropStart, programmingDropEnd].forEach((input) => input?.addEventListener("input", queueProgrammingDropScheduleCheck));
programmingRecurrenceModal?.querySelectorAll("[data-recurrence-scope]").forEach((button) => button.addEventListener("click", () => applyProgrammingRecurrenceScope(button.dataset.recurrenceScope)));
programmingRecurrenceClose?.addEventListener("click", () => closeProgrammingRecurrenceModal());
programmingRecurrenceCancel?.addEventListener("click", () => closeProgrammingRecurrenceModal());
programmingRecurrenceModal?.addEventListener("click", (event) => { if (event.target === programmingRecurrenceModal) closeProgrammingRecurrenceModal(); });
programmingPreviousWeek?.addEventListener("click", () => { programmingWeekOffset -= 1; renderProgrammingPage(); if (!programmingWeekPicker.hidden) renderProgrammingWeekPicker(); });
programmingNextWeek?.addEventListener("click", () => { programmingWeekOffset += 1; renderProgrammingPage(); if (!programmingWeekPicker.hidden) renderProgrammingWeekPicker(); });
programmingWeekLabel?.addEventListener("click", toggleProgrammingWeekPicker);
programmingPreviousMonth?.addEventListener("click", () => { programmingWeekPickerMonth.setMonth(programmingWeekPickerMonth.getMonth() - 1); renderProgrammingWeekPicker(); });
programmingNextMonth?.addEventListener("click", () => { programmingWeekPickerMonth.setMonth(programmingWeekPickerMonth.getMonth() + 1); renderProgrammingWeekPicker(); });
programmingPickerToday?.addEventListener("click", () => setProgrammingWeek(new Date()));
programmingPublish?.addEventListener("click", publishProgrammingPage);
programmingManageChannel?.addEventListener("click", () => {
  const channel = selectedProgrammingChannel();
  if (channel) editChannel(channel);
});
programmingNewCollection?.addEventListener("click", () => {
  setView("collections");
  openCollectionEditor();
});
cancelDeleteRule?.addEventListener("click", closeDeleteRuleModal);
cancelDeleteRuleX?.addEventListener("click", closeDeleteRuleModal);
confirmDeleteRule?.addEventListener("click", deletePendingRule);
deleteRuleModal?.addEventListener("click", (event) => {
  if (event.target === deleteRuleModal) {
    closeDeleteRuleModal();
  }
});
cancelDeleteChannel?.addEventListener("click", closeDeleteChannelModal);
cancelDeleteChannelX?.addEventListener("click", closeDeleteChannelModal);
confirmDeleteChannel?.addEventListener("click", deletePendingChannel);
deleteChannelModal?.addEventListener("click", (event) => {
  if (event.target === deleteChannelModal) {
    closeDeleteChannelModal();
  }
});
scanErrorClose?.addEventListener("click", closeScanErrorModal);
scanErrorModal?.addEventListener("click", (event) => {
  if (event.target === scanErrorModal) {
    closeScanErrorModal();
  }
});
document.addEventListener("keydown", (event) => {
  if (event.key === "Escape" && deleteRuleModal?.classList.contains("open")) {
    closeDeleteRuleModal();
  }
  if (event.key === "Escape" && deleteChannelModal?.classList.contains("open")) {
    closeDeleteChannelModal();
  }
  if (event.key === "Escape" && scanErrorModal?.classList.contains("open")) {
    closeScanErrorModal();
  }
  if (event.key === "Escape" && programmingBlockModal?.classList.contains("open")) {
    closeProgrammingBlockModal();
  }
  if (event.key === "Escape" && programmingDropModal?.classList.contains("open")) {
    closeProgrammingDropModal();
  }
  if (event.key === "Escape" && programmingRecurrenceModal?.classList.contains("open")) {
    closeProgrammingRecurrenceModal();
  }
  if (event.key === "Escape" && !programmingWeekPicker?.hidden) {
    closeProgrammingWeekPicker();
    programmingWeekLabel.focus();
  }
});
document.addEventListener("pointerdown", (event) => {
  if (!programmingWeekPicker?.hidden && !event.target.closest(".programming-week-navigator")) closeProgrammingWeekPicker();
});
navItems.forEach((item) => {
  item.addEventListener("click", (event) => {
    event.preventDefault();
    if (item.dataset.view === "movies") {
      movieLibraryMode = "all";
      libraryFilterLinks.forEach((link) => link.classList.remove("active"));
    }
    setView(item.dataset.view);
    if (item.dataset.view === "movies") {
      moviePage = 1;
      render();
    }
  });
});
document.querySelectorAll("[data-view-target]").forEach((button) => {
  button.addEventListener("click", () => setView(button.dataset.viewTarget));
});
function updateDashboardClock() {
  if (dashboardClock) dashboardClock.textContent = new Date().toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" });
}
updateDashboardClock();
window.setInterval(updateDashboardClock, 1000);
setView("dashboard");
loadSettings();
loadMetadataFacets();
loadMovies();
loadTVShows();
loadCollections();
loadChannels();
loadDashboardStorage();
pollChannelImportStatus();
render();
renderTVShows();
