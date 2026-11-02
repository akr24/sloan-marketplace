import { useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { listings, type Listing } from '../api/client'
import { useAuth } from '../auth/context'
import ListingForm from '../components/ListingForm'

export default function EditListingPage() {
  const { id } = useParams()
  const navigate = useNavigate()
  const { user } = useAuth()
  const [listing, setListing] = useState<Listing | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    listings
      .get(Number(id))
      .then(setListing)
      .catch((err: Error) => setError(err.message))
  }, [id])

  if (error) return <p className="error">{error === 'not found' ? 'That listing no longer exists.' : error}</p>
  if (!listing) return <p className="center muted">Loading…</p>
  if (listing.seller_id !== user?.id) return <p className="error">You can only edit your own listings.</p>

  return (
    <>
      <header className="page-head">
        <h1>Edit listing</h1>
        <Link to={`/listings/${listing.id}`} className="aside">
          ‹ Listing
        </Link>
      </header>
      <ListingForm
        initial={listing}
        submitLabel="Save changes"
        busyLabel="Saving…"
        onSubmit={async (body) => {
          await listings.update(listing.id, body)
          navigate(`/listings/${listing.id}`)
        }}
      />
    </>
  )
}
