type element
type nodeList

@val external document: {..} = "document"
@send external querySelectorAll: ({..}, string) => nodeList = "querySelectorAll"
@val external toArray: nodeList => array<element> = "Array.from"
@send external getAttribute: (element, string) => Nullable.t<string> = "getAttribute"
@set external setTextContent: (element, string) => unit = "textContent"
@val external navigatorLanguage: string = "navigator.language"

type intlNumberFormat
@new external makeFormatter: (string, {..}) => intlNumberFormat = "Intl.NumberFormat"
@send external format: (intlNumberFormat, float) => string = "format"
