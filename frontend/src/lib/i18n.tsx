import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'

export type Language = 'zh' | 'en'

type Dictionary = {
  appEyebrow: string
  heroTitle: string
  heroCopy: string
  socket: string
  job: string
  chain: string
  window: string
  activeProtocols: string
  scanSetup: string
  scanSetupTitle: string
  startScan: string
  waitingConnection: string
  scanInProgress: string
  selectOneProtocol: string
  startButtonHintConnected: string
  startButtonHintDisconnected: string
  startButtonHintRunning: string
  startButtonHintSelectProtocol: string
  chainId: string
  startBlock: string
  endBlock: string
  enableTraceVerification: string
  totals: string
  jobOverview: string
  jobOverviewCopy: string
  protocolsCompleted: string
  protocolRuntime: string
  protocolRuntimeCopy: string
  liveFindings: string
  newestOnTop: string
  liveLog: string
  runtimeSignals: string
  waitingForEvents: string
  findingsWaiting: string
  noFindingYet: string
  txHash: string
  protocol: string
  block: string
  blockProgress: string
  candidate: string
  verified: string
  strict: string
  yes: string
  no: string
  txUnit: string
  backToConsole: string
  loadingTransactionDetail: string
  transactionDetail: string
  detailOverview: string
  detailOverviewCopy: string
  protocolEvidence: string
  protocolEvidenceCopy: string
  protocolCountLabel: string
  interactionCountLabel: string
  strictInteractionCountLabel: string
  finalVerdictLabel: string
  keyAddresses: string
  keyAddressesCopy: string
  addressRoles: string
  noAddressData: string
  verificationEvidence: string
  verificationEvidenceCopy: string
  detectionConclusion: string
  detectionConclusionCopy: string
  keyReasons: string
  addressGraph: string
  addressGraphCopy: string
  sequenceDiagram: string
  sequenceDiagramCopy: string
  traceEvidence: string
  traceEvidenceCopy: string
  callChain: string
  callChainCopy: string
  traceStatus: string
  traceStatusAvailable: string
  traceStatusUnavailable: string
  traceStatusError: string
  callbackFrames: string
  callbackSubtree: string
  repaymentFrames: string
  traceUnavailable: string
  traceError: string
  traceErrorDetail: string
  noTraceFrames: string
  transactionHub: string
  identifiedFlows: string
  interactionEvidence: string
  interactionEvidenceCopy: string
  processTimeline: string
  processTimelineCopy: string
  timelineEntrypoint: string
  timelineAssetLeg: string
  timelineEvidence: string
  callbackSeenLabel: string
  settlementSeenLabel: string
  repaymentSeenLabel: string
  debtOpeningLabel: string
  exclusionReason: string
  methodSelector: string
  assetFlow: string
  interactionSummary: string
  asset: string
  role: string
  borrowedAmount: string
  repaidAmount: string
  premiumFee: string
  settlementModeLabel: string
  strictEvent: string
  noReasons: string
  roleInitiator: string
  roleOnBehalfOf: string
  roleReceiver: string
  roleCallbackTarget: string
  roleProvider: string
  roleFactory: string
  rolePair: string
  roleRepaymentTarget: string
  notAvailable: string
  provider: string
  receiver: string
  callbackTarget: string
  verificationNotes: string
  leg: string
  candidateLevel: string
  unverified: string
  nonStrict: string
  borrowed: string
  repaid: string
  premium: string
  fee: string
  strictLeg: string
  nonStrictLeg: string
  eventSeen: string
  eventMissing: string
  protocols: Record<string, string>
  statuses: Record<string, string>
}

type I18nValue = {
  language: Language
  setLanguage: (language: Language) => void
  t: Dictionary
  protocolName: (protocol: string) => string
  statusLabel: (status: string) => string
  boolLabel: (value: boolean) => string
  logMessage: (message: string, protocol?: string) => string
}

const STORAGE_KEY = 'scan-console-language'

const dictionaries: Record<Language, Dictionary> = {
  zh: {
    appEyebrow: '闪电贷扫描器',
    heroTitle: '三协议实时扫描控制台',
    heroCopy: '一次点击同时启动 Aave V3、Balancer V2 和 Uniswap V2，扫描过程中的进度与发现结果会实时写入控制台。',
    socket: '连接',
    job: '任务',
    chain: '链',
    window: '扫描窗口',
    activeProtocols: '启用协议',
    scanSetup: '扫描配置',
    scanSetupTitle: '实时闪电贷扫描',
    startScan: '开始扫描',
    waitingConnection: '等待连接',
    scanInProgress: '扫描进行中',
    selectOneProtocol: '至少选择一个协议',
    startButtonHintConnected: '后端连接正常，可以开始扫描。',
    startButtonHintDisconnected: '正在等待与后端建立连接，请先确认 8082 服务已启动。',
    startButtonHintRunning: '当前已有扫描任务在运行，请等待这一轮完成。',
    startButtonHintSelectProtocol: '请至少选择一个协议后再开始扫描。',
    chainId: '链 ID',
    startBlock: '起始区块',
    endBlock: '结束区块',
    enableTraceVerification: '启用 Trace 验证',
    totals: '总览统计',
    jobOverview: '任务总览',
    jobOverviewCopy: '当前任务的累计结果与完成进度会在这里持续刷新。',
    protocolsCompleted: '已完成协议数',
    protocolRuntime: '协议运行态',
    protocolRuntimeCopy: '三协议共享同一任务窗口，但分别汇报进度、命中与错误。',
    liveFindings: '实时发现',
    newestOnTop: '最新结果置顶',
    liveLog: '实时日志',
    runtimeSignals: '运行信号',
    waitingForEvents: '等待扫描事件。',
    findingsWaiting: '扫描器发现结果后会立即出现在这里。',
    noFindingYet: '暂无结果',
    txHash: '交易哈希',
    protocol: '协议',
    block: '区块',
    blockProgress: '区块进度',
    candidate: '初筛命中',
    verified: '验证通过',
    strict: '严格通过',
    yes: '是',
    no: '否',
    txUnit: '笔',
    backToConsole: '返回控制台',
    loadingTransactionDetail: '正在加载交易详情...',
    transactionDetail: '交易总览',
    detailOverview: '协议识别证据',
    detailOverviewCopy: '先确认这笔交易涉及哪些协议、识别到多少交互以及最终判定，再往下看地址和证据。',
    protocolEvidence: '协议识别证据',
    protocolEvidenceCopy: '汇总协议命中范围、交互规模和最终判定，先回答“这笔交易为什么值得继续看”。',
    protocolCountLabel: '涉及协议',
    interactionCountLabel: '识别交互',
    strictInteractionCountLabel: '严格交互',
    finalVerdictLabel: '扫描结论',
    keyAddresses: '关键参与地址',
    keyAddressesCopy: '把同一地址按扫描器识别到的角色合并，便于快速理解这笔交易里的参与方。',
    addressRoles: '地址角色',
    noAddressData: '当前没有可归类的地址。',
    verificationEvidence: '规则验证证据',
    verificationEvidenceCopy: '按协议交互汇总规则验证证据，直接说明为什么通过、卡住或被排除。',
    detectionConclusion: '扫描器结论',
    detectionConclusionCopy: '把当前详情页已有证据整理成人话，便于答辩时快速说明为什么命中。',
    keyReasons: '关键依据',
    addressGraph: '关键地址关系图',
    addressGraphCopy: '用关键地址和资产腿概括这笔交易中“谁参与了、资产往哪里走”。',
    sequenceDiagram: '轻量时序图',
    sequenceDiagramCopy: '按入口、借出、归还和验证顺序展示扫描器识别到的交易过程。',
    traceEvidence: '调用路径证据',
    traceEvidenceCopy: '基于 callTracer 汇总回调路径、归还路径和严格证据，说明调用链为什么支持当前判定。',
    callChain: '完整内部调用链',
    callChainCopy: '按调用深度展开完整 trace，并突出 callback 子树、repayment 路径和报错帧。',
    traceStatus: 'Trace 状态',
    traceStatusAvailable: '可用',
    traceStatusUnavailable: '不可用',
    traceStatusError: '拉取失败',
    callbackFrames: '回调帧',
    callbackSubtree: '回调内部路径',
    repaymentFrames: '归还路径帧',
    traceUnavailable: '当前 RPC 没有提供 trace 数据。',
    traceError: 'trace 拉取失败',
    traceErrorDetail: '当前 RPC 可能不支持历史交易追踪，也可能是限流或历史状态不可用。',
    noTraceFrames: '当前没有可展示的 trace 调用链。',
    transactionHub: '交易枢纽',
    identifiedFlows: '识别到的资产流',
    interactionEvidence: '交互识别证据',
    interactionEvidenceCopy: '按协议交互展开原始识别结果，便于对照扫描器究竟识别到了哪些 protocol-level interaction。',
    processTimeline: '过程时间线',
    processTimelineCopy: '按 interaction 顺序展示入口、资产腿和验证信号，强调扫描器如何理解这笔交易。',
    timelineEntrypoint: '进入协议入口',
    timelineAssetLeg: '资产腿',
    timelineEvidence: '验证信号',
    callbackSeenLabel: '回调命中',
    settlementSeenLabel: '结算命中',
    repaymentSeenLabel: '归还命中',
    debtOpeningLabel: '存在开债',
    exclusionReason: '排除原因',
    methodSelector: '方法选择器',
    assetFlow: '资产记录证据',
    interactionSummary: '协议交互明细',
    asset: '资产',
    role: '角色',
    borrowedAmount: '借出数量',
    repaidAmount: '归还数量',
    premiumFee: '溢价 / 手续费',
    settlementModeLabel: '结算模式',
    strictEvent: '严格 / 事件',
    noReasons: '暂无额外说明。',
    roleInitiator: '发起者',
    roleOnBehalfOf: '代偿关系',
    roleReceiver: '回调接收者',
    roleCallbackTarget: '回调目标',
    roleProvider: '流动性提供方',
    roleFactory: '工厂合约',
    rolePair: '交易对合约',
    roleRepaymentTarget: '归还目标',
    notAvailable: '暂无',
    provider: '提供方',
    receiver: '接收方',
    callbackTarget: '回调目标',
    verificationNotes: '验证说明',
    leg: '资产腿',
    candidateLevel: '初筛级别',
    unverified: '未验证',
    nonStrict: '非严格',
    borrowed: '借出',
    repaid: '归还',
    premium: '溢价',
    fee: '手续费',
    strictLeg: '严格资产腿',
    nonStrictLeg: '非严格资产腿',
    eventSeen: '事件已命中',
    eventMissing: '事件缺失',
    protocols: {
      aave_v3: 'Aave V3',
      balancer_v2: 'Balancer V2',
      uniswap_v2: 'Uniswap V2',
    },
    statuses: {
      idle: '未开始',
      pending: '待启动',
      running: '运行中',
      completed: '已完成',
      failed: '失败',
      connected: '已连接',
      connecting: '连接中',
      disconnected: '未连接',
    },
  },
  en: {
    appEyebrow: 'Flashloan Scanner',
    heroTitle: 'Three-Protocol Live Console',
    heroCopy: 'One click starts Aave V3, Balancer V2, and Uniswap V2 together. Progress and findings stream into the console while the scan runs.',
    socket: 'Socket',
    job: 'Job',
    chain: 'Chain',
    window: 'Scan Window',
    activeProtocols: 'Active Protocols',
    scanSetup: 'Scan Setup',
    scanSetupTitle: 'Real-time flashloan scan',
    startScan: 'Start Scan',
    waitingConnection: 'Waiting for Connection',
    scanInProgress: 'Scan in Progress',
    selectOneProtocol: 'Select One Protocol',
    startButtonHintConnected: 'Backend is connected and ready to start a scan.',
    startButtonHintDisconnected: 'Waiting for backend connection. Make sure the service on port 8082 is running.',
    startButtonHintRunning: 'A scan job is already running. Wait for it to finish before starting another one.',
    startButtonHintSelectProtocol: 'Choose at least one protocol before starting a scan.',
    chainId: 'Chain ID',
    startBlock: 'Start Block',
    endBlock: 'End Block',
    enableTraceVerification: 'Enable trace verification',
    totals: 'Totals',
    jobOverview: 'Job Overview',
    jobOverviewCopy: 'Rolling totals and completion for the current job update here in real time.',
    protocolsCompleted: 'Protocols completed',
    protocolRuntime: 'Protocol Runtime',
    protocolRuntimeCopy: 'Protocols share one scan window but report progress, hits, and failures independently.',
    liveFindings: 'Live Findings',
    newestOnTop: 'Newest results on top',
    liveLog: 'Live Log',
    runtimeSignals: 'Runtime signals',
    waitingForEvents: 'Waiting for scan events.',
    findingsWaiting: 'Findings will stream here as soon as the scanner emits them.',
    noFindingYet: 'No finding yet',
    txHash: 'Tx Hash',
    protocol: 'Protocol',
    block: 'Block',
    blockProgress: 'Block Progress',
    candidate: 'Initial Hits',
    verified: 'Verified',
    strict: 'Strict Passed',
    yes: 'Yes',
    no: 'No',
    txUnit: 'tx',
    backToConsole: 'Back to console',
    loadingTransactionDetail: 'Loading transaction detail...',
    transactionDetail: 'Transaction Summary',
    detailOverview: 'Protocol Identification Evidence',
    detailOverviewCopy: 'Confirm the involved protocols, identified interactions, and final verdict first, then inspect addresses and evidence.',
    protocolEvidence: 'Protocol Identification Evidence',
    protocolEvidenceCopy: 'Summarize protocol matches, interaction scale, and the final verdict so the page first answers why this transaction matters.',
    protocolCountLabel: 'Protocols Involved',
    interactionCountLabel: 'Interactions Found',
    strictInteractionCountLabel: 'Strict Interactions',
    finalVerdictLabel: 'Verdict',
    keyAddresses: 'Key Participants',
    keyAddressesCopy: 'Merge repeated addresses by scanner-recognized role so the participants in this transaction are easier to read.',
    addressRoles: 'Roles',
    noAddressData: 'No classified addresses found.',
    verificationEvidence: 'Rule Verification Evidence',
    verificationEvidenceCopy: 'Summarize rule-level evidence per protocol interaction so it is clear why a result passed, stalled, or was excluded.',
    detectionConclusion: 'Scanner Conclusion',
    detectionConclusionCopy: 'Turn the current evidence into a short human-readable verdict for demos and reviews.',
    keyReasons: 'Key Reasons',
    addressGraph: 'Address Graph',
    addressGraphCopy: 'Summarize who participated in the transaction and where the detected asset legs moved.',
    sequenceDiagram: 'Sequence Diagram',
    sequenceDiagramCopy: 'Show the detected entrypoint, borrow, repay, and evidence steps in order.',
    traceEvidence: 'Call Path Evidence',
    traceEvidenceCopy: 'Aggregate callback-path, repayment-path, and strict evidence from callTracer to explain why the call chain supports the verdict.',
    callChain: 'Full Internal Call Chain',
    callChainCopy: 'Expand the complete trace by call depth and highlight callback subtrees, repayment paths, and error frames.',
    traceStatus: 'Trace Status',
    traceStatusAvailable: 'available',
    traceStatusUnavailable: 'unavailable',
    traceStatusError: 'fetch failed',
    callbackFrames: 'Callback Frames',
    callbackSubtree: 'Callback Path',
    repaymentFrames: 'Repayment Frames',
    traceUnavailable: 'This RPC did not return trace data.',
    traceError: 'trace fetch failed',
    traceErrorDetail: 'The current RPC may not support historical tracing, or the request may have been rate-limited.',
    noTraceFrames: 'No trace call chain is available for this transaction.',
    transactionHub: 'Transaction Hub',
    identifiedFlows: 'Detected Asset Flows',
    interactionEvidence: 'Interaction Identification Evidence',
    interactionEvidenceCopy: 'Expand the raw protocol-level interactions so it is clear what the scanner actually identified in this transaction.',
    processTimeline: 'Process Timeline',
    processTimelineCopy: 'Show entrypoint, asset legs, and evidence in interaction order so the scanner logic is easier to follow.',
    timelineEntrypoint: 'Entrypoint',
    timelineAssetLeg: 'Asset Leg',
    timelineEvidence: 'Evidence',
    callbackSeenLabel: 'Callback Seen',
    settlementSeenLabel: 'Settlement Seen',
    repaymentSeenLabel: 'Repayment Seen',
    debtOpeningLabel: 'Debt Opening',
    exclusionReason: 'Exclusion Reason',
    methodSelector: 'Method Selector',
    assetFlow: 'Asset Record Evidence',
    interactionSummary: 'Protocol Interaction Details',
    asset: 'Asset',
    role: 'Role',
    borrowedAmount: 'Borrowed',
    repaidAmount: 'Repaid',
    premiumFee: 'Premium / Fee',
    settlementModeLabel: 'Settlement',
    strictEvent: 'Strict / Event',
    noReasons: 'No additional reasons.',
    roleInitiator: 'Initiator',
    roleOnBehalfOf: 'On Behalf Of',
    roleReceiver: 'Receiver',
    roleCallbackTarget: 'Callback Target',
    roleProvider: 'Provider',
    roleFactory: 'Factory',
    rolePair: 'Pair',
    roleRepaymentTarget: 'Repayment Target',
    notAvailable: 'N/A',
    provider: 'Provider',
    receiver: 'Receiver',
    callbackTarget: 'Callback Target',
    verificationNotes: 'Verification Notes',
    leg: 'Leg',
    candidateLevel: 'Candidate level',
    unverified: 'Unverified',
    nonStrict: 'Non-strict',
    borrowed: 'borrowed',
    repaid: 'repaid',
    premium: 'premium',
    fee: 'fee',
    strictLeg: 'strict leg',
    nonStrictLeg: 'non-strict leg',
    eventSeen: 'event seen',
    eventMissing: 'event missing',
    protocols: {
      aave_v3: 'Aave V3',
      balancer_v2: 'Balancer V2',
      uniswap_v2: 'Uniswap V2',
    },
    statuses: {
      idle: 'Idle',
      pending: 'Pending',
      running: 'Running',
      completed: 'Completed',
      failed: 'Failed',
      connected: 'Connected',
      connecting: 'Connecting',
      disconnected: 'Disconnected',
    },
  },
}

const I18nContext = createContext<I18nValue | undefined>(undefined)

export function I18nProvider({ children }: { children: ReactNode }) {
  const [language, setLanguage] = useState<Language>(() => {
    if (typeof window === 'undefined') {
      return 'zh'
    }
    return window.localStorage.getItem(STORAGE_KEY) === 'en' ? 'en' : 'zh'
  })

  useEffect(() => {
    window.localStorage.setItem(STORAGE_KEY, language)
  }, [language])

  const value = useMemo<I18nValue>(() => {
    const t = dictionaries[language]
    return {
      language,
      setLanguage,
      t,
      protocolName: (protocol: string) => t.protocols[protocol] ?? protocol,
      statusLabel: (status: string) => t.statuses[status] ?? status,
      boolLabel: (value: boolean) => (value ? t.yes : t.no),
      logMessage: (message: string, protocol?: string) => translateLogMessage(language, message, protocol),
    }
  }, [language])

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>
}

export function useI18n() {
  const context = useContext(I18nContext)
  if (!context) {
    throw new Error('useI18n must be used within I18nProvider')
  }
  return context
}

function translateLogMessage(language: Language, rawMessage: string, protocol?: string) {
  let message = rawMessage
  if (protocol) {
    const prefix = `${protocol} `
    if (message.startsWith(prefix)) {
      message = message.slice(prefix.length)
    }
  }

  if (language === 'en') {
    return message
  }

  if (message === 'scan job started') {
    return '扫描任务已启动'
  }
  if (message === 'scan job completed') {
    return '扫描任务已完成'
  }
  if (message.startsWith('scan job failed: ')) {
    return `扫描任务失败：${message.slice('scan job failed: '.length)}`
  }
  if (message === 'scan started') {
    return '扫描已启动'
  }
  if (message === 'scan completed') {
    return '扫描已完成'
  }
  if (message.startsWith('scan failed: ')) {
    return `扫描失败：${message.slice('scan failed: '.length)}`
  }
  if (message.startsWith('found tx ')) {
    return `发现交易 ${message.slice('found tx '.length)}`
  }

  return message
}
