import ListInput from './ListInput'

type Props = { value: string[]; onChange: (eans: string[]) => void }

/** EAN list editor: digits only, up to 13 characters per code. */
export default function EanListInput({ value, onChange }: Props) {
  return (
    <ListInput
      value={value}
      onChange={onChange}
      digitsOnly
      maxLength={13}
      placeholder="7891234567890"
      addLabel="+ adicionar outro EAN"
      label="EAN"
    />
  )
}
