'use client';

import { useState, useEffect } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import {
  Star,
  Search,
  Shield,
  Music,
  Video,
  Code,
  Gamepad2,
  Book,
  Briefcase,
  TrendingUp,
  Users,
  CreditCard,
  Wallet,
} from 'lucide-react';
import Link from 'next/link';

// Type for marketplace service
interface MarketplaceService {
  id: number;
  name: string;
  description: string;
  category: string;
  price: string;
  period: string;
  rating: number;
  reviews: number;
  icon: React.ComponentType<{ className?: string }>;
  image: string;
  features: string[];
  popular: boolean;
  provider: string;
  tags: string[];
}

// Mock data for marketplace services
const featuredServices: MarketplaceService[] = [
  {
    id: 1,
    name: 'Spotify Premium',
    description: 'Ad-free music streaming with offline downloads and high-quality audio',
    category: 'Entertainment',
    price: '$9.99',
    period: 'month',
    rating: 4.8,
    reviews: 1250,
    icon: Music,
    image: '/api/placeholder/300/200',
    features: ['Ad-free music', 'Offline downloads', 'High-quality audio', 'Unlimited skips'],
    popular: true,
    provider: 'Spotify Inc.',
    tags: ['Music', 'Streaming', 'Audio'],
  },
  {
    id: 2,
    name: 'Netflix Premium',
    description: 'Stream unlimited movies and TV shows in 4K Ultra HD',
    category: 'Entertainment',
    price: '$15.99',
    period: 'month',
    rating: 4.6,
    reviews: 2100,
    icon: Video,
    image: '/api/placeholder/300/200',
    features: ['4K Ultra HD', 'Multiple screens', 'Offline viewing', 'No ads'],
    popular: true,
    provider: 'Netflix Inc.',
    tags: ['Movies', 'TV Shows', 'Streaming'],
  },
  {
    id: 3,
    name: 'GitHub Pro',
    description: 'Advanced development tools and unlimited private repositories',
    category: 'Development',
    price: '$4.00',
    period: 'month',
    rating: 4.9,
    reviews: 850,
    icon: Code,
    image: '/api/placeholder/300/200',
    features: [
      'Unlimited private repos',
      'Advanced security',
      'Priority support',
      'GitHub Actions',
    ],
    popular: false,
    provider: 'GitHub Inc.',
    tags: ['Development', 'Code', 'Git'],
  },
];

const allServices = [
  ...featuredServices,
  {
    id: 4,
    name: 'Adobe Creative Cloud',
    description: 'Complete suite of creative applications for design and video editing',
    category: 'Design',
    price: '$52.99',
    period: 'month',
    rating: 4.7,
    reviews: 1800,
    icon: Briefcase,
    image: '/api/placeholder/300/200',
    features: ['Photoshop', 'Illustrator', 'Premiere Pro', 'After Effects', 'Cloud storage'],
    popular: false,
    provider: 'Adobe Inc.',
    tags: ['Design', 'Video', 'Creative'],
  },
  {
    id: 5,
    name: 'Notion Pro',
    description: 'All-in-one workspace for notes, tasks, wikis, and databases',
    category: 'Productivity',
    price: '$8.00',
    period: 'month',
    rating: 4.5,
    reviews: 920,
    icon: Book,
    image: '/api/placeholder/300/200',
    features: ['Unlimited blocks', 'Version history', 'Advanced permissions', 'API access'],
    popular: false,
    provider: 'Notion Labs Inc.',
    tags: ['Productivity', 'Notes', 'Collaboration'],
  },
  {
    id: 6,
    name: 'Discord Nitro',
    description: 'Enhanced Discord experience with better streaming and file sharing',
    category: 'Communication',
    price: '$9.99',
    period: 'month',
    rating: 4.4,
    reviews: 1100,
    icon: Users,
    image: '/api/placeholder/300/200',
    features: ['HD video streaming', 'Larger file uploads', 'Custom emojis', 'Server boosts'],
    popular: false,
    provider: 'Discord Inc.',
    tags: ['Communication', 'Gaming', 'Social'],
  },
  {
    id: 7,
    name: 'Cloudflare Pro',
    description: 'Enhanced website performance and security features',
    category: 'Infrastructure',
    price: '$20.00',
    period: 'month',
    rating: 4.6,
    reviews: 650,
    icon: Shield,
    image: '/api/placeholder/300/200',
    features: [
      'Advanced DDoS protection',
      'Image optimization',
      'Mobile optimization',
      'Priority support',
    ],
    popular: false,
    provider: 'Cloudflare Inc.',
    tags: ['Security', 'Performance', 'CDN'],
  },
  {
    id: 8,
    name: 'Steam Deck Game Pass',
    description: 'Access to hundreds of PC games with cloud gaming support',
    category: 'Gaming',
    price: '$14.99',
    period: 'month',
    rating: 4.3,
    reviews: 780,
    icon: Gamepad2,
    image: '/api/placeholder/300/200',
    features: ['100+ games', 'Cloud gaming', 'Offline play', 'New releases'],
    popular: false,
    provider: 'Valve Corporation',
    tags: ['Gaming', 'PC Games', 'Cloud'],
  },
];

const categories = [
  { id: 'all', name: 'All Services', icon: TrendingUp },
  { id: 'Entertainment', name: 'Entertainment', icon: Video },
  { id: 'Development', name: 'Development', icon: Code },
  { id: 'Design', name: 'Design', icon: Briefcase },
  { id: 'Productivity', name: 'Productivity', icon: Book },
  { id: 'Communication', name: 'Communication', icon: Users },
  { id: 'Infrastructure', name: 'Infrastructure', icon: Shield },
  { id: 'Gaming', name: 'Gaming', icon: Gamepad2 },
];

export default function CustomerMarketplacePage() {
  const [isClient, setIsClient] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedCategory, setSelectedCategory] = useState('all');
  const [sortBy, setSortBy] = useState('popular');
  const [filteredServices, setFilteredServices] = useState(allServices);

  useEffect(() => {
    setIsClient(true);
  }, []);

  useEffect(() => {
    let filtered = allServices;

    // Filter by category
    if (selectedCategory !== 'all') {
      filtered = filtered.filter((service) => service.category === selectedCategory);
    }

    // Filter by search query
    if (searchQuery) {
      filtered = filtered.filter(
        (service) =>
          service.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
          service.description.toLowerCase().includes(searchQuery.toLowerCase()) ||
          service.tags.some((tag) => tag.toLowerCase().includes(searchQuery.toLowerCase()))
      );
    }

    // Sort services
    switch (sortBy) {
      case 'popular':
        filtered.sort((a, b) => (b.popular ? 1 : 0) - (a.popular ? 1 : 0) || b.rating - a.rating);
        break;
      case 'price-low':
        filtered.sort(
          (a, b) => parseFloat(a.price.replace('$', '')) - parseFloat(b.price.replace('$', ''))
        );
        break;
      case 'price-high':
        filtered.sort(
          (a, b) => parseFloat(b.price.replace('$', '')) - parseFloat(a.price.replace('$', ''))
        );
        break;
      case 'rating':
        filtered.sort((a, b) => b.rating - a.rating);
        break;
      default:
        break;
    }

    setFilteredServices(filtered);
  }, [searchQuery, selectedCategory, sortBy]);

  const handleSubscribe = (service: MarketplaceService) => {
    // In a real app, this would handle the subscription process
    // For now, just show an alert
    alert(`Redirecting to subscription page for ${service.name}`);
  };

  if (!isClient) {
    return (
      <div className="container mx-auto p-8">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-purple-600 mx-auto"></div>
      </div>
    );
  }

  return (
    <div className="container mx-auto p-8 space-y-8">
      {/* Header */}
      <div className="text-center space-y-2">
        <h1 className="text-4xl font-bold">Service Marketplace</h1>
        <p className="text-lg text-muted-foreground">
          Discover and subscribe to services with cryptocurrency payments
        </p>
      </div>

      {/* Search and Filters */}
      <div className="flex flex-col md:flex-row gap-4 items-center">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search services..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-10"
          />
        </div>
        <Select value={selectedCategory} onValueChange={setSelectedCategory}>
          <SelectTrigger className="w-full md:w-48">
            <SelectValue placeholder="Category" />
          </SelectTrigger>
          <SelectContent>
            {categories.map((category) => (
              <SelectItem key={category.id} value={category.id}>
                <div className="flex items-center gap-2">
                  <category.icon className="h-4 w-4" />
                  {category.name}
                </div>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select value={sortBy} onValueChange={setSortBy}>
          <SelectTrigger className="w-full md:w-48">
            <SelectValue placeholder="Sort by" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="popular">Most Popular</SelectItem>
            <SelectItem value="rating">Highest Rated</SelectItem>
            <SelectItem value="price-low">Price: Low to High</SelectItem>
            <SelectItem value="price-high">Price: High to Low</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <Tabs defaultValue="featured" className="w-full">
        <TabsList className="grid w-full grid-cols-2">
          <TabsTrigger value="featured">Featured Services</TabsTrigger>
          <TabsTrigger value="all">All Services</TabsTrigger>
        </TabsList>

        <TabsContent value="featured" className="space-y-6">
          {/* Featured Services Hero */}
          <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
            {featuredServices.map((service) => {
              const IconComponent = service.icon;
              return (
                <Card
                  key={service.id}
                  className="relative overflow-hidden border-2 border-purple-200 dark:border-purple-800"
                >
                  <div className="absolute top-4 right-4">
                    <Badge
                      variant="secondary"
                      className="bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200"
                    >
                      Featured
                    </Badge>
                  </div>
                  <CardHeader className="pb-4">
                    <div className="flex items-center gap-3">
                      <div className="p-2 bg-purple-100 dark:bg-purple-900 rounded-lg">
                        <IconComponent className="h-6 w-6 text-purple-600 dark:text-purple-400" />
                      </div>
                      <div className="flex-1">
                        <CardTitle className="text-xl">{service.name}</CardTitle>
                        <div className="flex items-center gap-2 mt-1">
                          <div className="flex items-center gap-1">
                            <Star className="h-4 w-4 fill-yellow-400 text-yellow-400" />
                            <span className="text-sm font-medium">{service.rating}</span>
                          </div>
                          <span className="text-sm text-muted-foreground">
                            ({service.reviews} reviews)
                          </span>
                        </div>
                      </div>
                    </div>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <CardDescription className="text-sm">{service.description}</CardDescription>

                    <div className="space-y-2">
                      <div className="flex items-center justify-between">
                        <span className="text-sm font-medium">Features:</span>
                        <Badge variant="outline">{service.category}</Badge>
                      </div>
                      <div className="flex flex-wrap gap-1">
                        {service.features.slice(0, 3).map((feature, idx) => (
                          <Badge key={idx} variant="secondary" className="text-xs">
                            {feature}
                          </Badge>
                        ))}
                        {service.features.length > 3 && (
                          <Badge variant="secondary" className="text-xs">
                            +{service.features.length - 3} more
                          </Badge>
                        )}
                      </div>
                    </div>

                    <div className="flex items-center justify-between pt-4 border-t">
                      <div className="text-left">
                        <div className="text-2xl font-bold text-purple-600">{service.price}</div>
                        <div className="text-sm text-muted-foreground">per {service.period}</div>
                      </div>
                      <Button
                        onClick={() => handleSubscribe(service)}
                        className="bg-purple-600 hover:bg-purple-700"
                      >
                        Subscribe Now
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              );
            })}
          </div>
        </TabsContent>

        <TabsContent value="all" className="space-y-6">
          {/* All Services Grid */}
          <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
            {filteredServices.map((service) => {
              const IconComponent = service.icon;
              return (
                <Card
                  key={service.id}
                  className="relative overflow-hidden hover:shadow-lg transition-shadow"
                >
                  {service.popular && (
                    <div className="absolute top-4 right-4">
                      <Badge
                        variant="secondary"
                        className="bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200"
                      >
                        Popular
                      </Badge>
                    </div>
                  )}
                  <CardHeader className="pb-4">
                    <div className="flex items-center gap-3">
                      <div className="p-2 bg-gray-100 dark:bg-gray-800 rounded-lg">
                        <IconComponent className="h-6 w-6 text-gray-600 dark:text-gray-400" />
                      </div>
                      <div className="flex-1">
                        <CardTitle className="text-lg">{service.name}</CardTitle>
                        <div className="flex items-center gap-2 mt-1">
                          <div className="flex items-center gap-1">
                            <Star className="h-4 w-4 fill-yellow-400 text-yellow-400" />
                            <span className="text-sm font-medium">{service.rating}</span>
                          </div>
                          <span className="text-sm text-muted-foreground">({service.reviews})</span>
                        </div>
                      </div>
                    </div>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <CardDescription className="text-sm line-clamp-2">
                      {service.description}
                    </CardDescription>

                    <div className="flex items-center justify-between">
                      <Badge variant="outline">{service.category}</Badge>
                      <span className="text-sm text-muted-foreground">by {service.provider}</span>
                    </div>

                    <div className="flex flex-wrap gap-1">
                      {service.tags.map((tag, idx) => (
                        <Badge key={idx} variant="secondary" className="text-xs">
                          {tag}
                        </Badge>
                      ))}
                    </div>

                    <div className="flex items-center justify-between pt-4 border-t">
                      <div className="text-left">
                        <div className="text-xl font-bold">{service.price}</div>
                        <div className="text-sm text-muted-foreground">per {service.period}</div>
                      </div>
                      <Button onClick={() => handleSubscribe(service)} variant="outline" size="sm">
                        Subscribe
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              );
            })}
          </div>

          {filteredServices.length === 0 && (
            <div className="text-center py-12">
              <div className="text-muted-foreground">
                <Search className="h-12 w-12 mx-auto mb-4" />
                <p className="text-lg">No services found</p>
                <p className="text-sm">Try adjusting your search or filter criteria</p>
              </div>
            </div>
          )}
        </TabsContent>
      </Tabs>

      {/* Categories Overview */}
      <div className="space-y-4">
        <h2 className="text-2xl font-bold">Browse by Category</h2>
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          {categories
            .filter((cat) => cat.id !== 'all')
            .map((category) => {
              const IconComponent = category.icon;
              const serviceCount = allServices.filter((s) => s.category === category.name).length;
              return (
                <Card
                  key={category.id}
                  className="hover:shadow-md transition-shadow cursor-pointer"
                  onClick={() => setSelectedCategory(category.id)}
                >
                  <CardContent className="p-6 text-center">
                    <IconComponent className="h-8 w-8 mx-auto mb-2 text-purple-600" />
                    <h3 className="font-semibold">{category.name}</h3>
                    <p className="text-sm text-muted-foreground">
                      {serviceCount} service{serviceCount !== 1 ? 's' : ''}
                    </p>
                  </CardContent>
                </Card>
              );
            })}
        </div>
      </div>

      {/* Call to Action */}
      <Card className="bg-gradient-to-r from-purple-50 to-blue-50 dark:from-purple-900/20 dark:to-blue-900/20 border-purple-200 dark:border-purple-800">
        <CardContent className="p-8 text-center">
          <h2 className="text-2xl font-bold mb-2">Ready to Get Started?</h2>
          <p className="text-muted-foreground mb-4">
            Subscribe to your favorite services and pay with cryptocurrency through your Web3 wallet
          </p>
          <div className="flex flex-col sm:flex-row gap-4 justify-center">
            <Button asChild className="bg-purple-600 hover:bg-purple-700">
              <Link href="/customers/wallet">
                <Wallet className="mr-2 h-4 w-4" />
                Check Wallet Balance
              </Link>
            </Button>
            <Button asChild variant="outline">
              <Link href="/customers/subscriptions">
                <CreditCard className="mr-2 h-4 w-4" />
                Manage Subscriptions
              </Link>
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
