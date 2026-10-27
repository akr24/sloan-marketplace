import { Link, useNavigate } from 'react-router-dom'
import { listings } from '../api/client'
import ListingForm from '../components/ListingForm'

// The in-app fallback for sellers who'd rather not use the bot.
export default function NewListingPage() {
  const navigate = useNavigate()
  return (
    <>
      <header className="page-head">
        <h1>Sell tickets</h1>
        <Link to="/sell" className="aside">
          ‹ Sell
        </Link>
      </header>
      <ListingForm
        submitLabel="Publish listing"
        busyLabel="Publishing…"
        onSubmit={async (body) => {
          const l = await listings.create(body)
          navigate(`/listings/${l.id}/live`, { replace: true })
        }}
      />
    </>
  )
}
