export default function ({
  type = 'search',
  value,
  placeholder,
  handleInputChange,
  handleKeyDown,
  disabled = false,
  error,
  hasDropdownSelection = true,
  optionType,
  options,
  handleSelectOption,
  selectedItem
}) {
  return (
    <div>
      <div className={`relative w-full pr-4 py-3 border-b bg-transparent focus:outline-none focus:ring focus:ring-cyan-600 disabled:opacity-30 ${error ? 'border-pink-500' : 'border-gray-800'}`}>
        <input
          autoComplete='off'
          spellCheck='false'
          autoFocus
          type={type}
          value={value}
          className={`block ${hasDropdownSelection ? 'w-5/6' : 'w-full'} sm:text-2xs bg-transparent focus:outline-none placeholder:italic placeholder:text-gray-400 placeholder:text-2xs`}
          placeholder={placeholder}
          onChange={handleInputChange}
          onKeyDown={handleKeyDown}
          disabled={disabled}
        />
        {hasDropdownSelection &&
          <div className='absolute inset-y-0 right-2 flex items-center'>
            <label htmlFor={optionType} className='sr-only'>
              {optionType}
            </label>
            <select
              id={optionType}
              name={optionType}
              onChange={handleSelectOption}
              value={selectedItem}
              className='h-full py-0 pl-2 border-transparent bg-transparent text-2xs text-gray-400 focus:outline-none'
            >
              {options.map((option) => (
                <option key={option} value={option}>{option}</option>
              ))}
            </select>
          </div>}
      </div>
    </div>
  )
}
