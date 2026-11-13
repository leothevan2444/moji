import { FieldLabel } from "./SettingsTabCommon";
export default function DownloaderSwitch({label,info,checked,onChange,disabled=false}:{label:string;info?:string;checked:boolean;onChange:(value:boolean)=>void;disabled?:boolean}) {
 return <div className="switch-row"><span className="switch-row__label"><FieldLabel text={label} info={info}/></span><label className="switch" role="switch" aria-checked={checked} aria-disabled={disabled}><input type="checkbox" aria-label={label} checked={checked} disabled={disabled} onChange={event=>onChange(event.target.checked)}/><span className="switch__track" aria-hidden="true"/><span className="switch__thumb" aria-hidden="true"/></label></div>;
}
