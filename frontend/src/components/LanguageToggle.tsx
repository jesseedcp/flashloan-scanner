import { useI18n, type Language } from '../lib/i18n'

export function LanguageToggle() {
  const { language, setLanguage } = useI18n()

  return (
    <div className="language-toggle">
      <LanguageButton label="中文" value="zh" active={language === 'zh'} onClick={setLanguage} />
      <LanguageButton label="EN" value="en" active={language === 'en'} onClick={setLanguage} />
    </div>
  )
}

function LanguageButton({
  label,
  value,
  active,
  onClick,
}: {
  label: string
  value: Language
  active: boolean
  onClick: (value: Language) => void
}) {
  return (
    <button className={`language-button ${active ? 'active' : ''}`} onClick={() => onClick(value)} type="button">
      {label}
    </button>
  )
}
