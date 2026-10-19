import { Link, useNavigate } from 'react-router-dom'
import { listings } from '../api/client'
import ListingForm from '../components/ListingForm'

export default function SellPage() {
  const navigate = useNavigate()
  return (
    <>
      <h2>Sell tickets</h2>
      <ListingForm
        submitLabel="Post listing"
        busyLabel="Posting…"
        onSubmit={async (body) => {
          const l = await listings.create(body)
          navigate(`/listings/${l.id}?posted`)
        }}
      />
      <p className="muted small center">
        Rather post from WhatsApp? <Link to="/account">Here's how</Link>.
      </p>
    </>
  )
}
