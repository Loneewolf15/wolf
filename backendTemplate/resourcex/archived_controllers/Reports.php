<?php
class Products extends Controller
{

    public function __construct()
    {
        $this->shopModel = $this->model('Shop');
        $this->userModel = $this->model('User');
        $this->productModel = $this->model('Product');
         $this->serverKey  = 'secret_server_key';
    }

    public function createProduct()
    {
        try {
            $userData = $this->RouteProtection();
        } catch (UnexpectedValueException $e) {
            $res = [
                'status' => 401,
                'message' =>  $e->getMessage(),
            ];
            print_r(json_encode($res));
            exit;
        }
        if ($_SERVER['REQUEST_METHOD'] == 'POST') {
             $shopDetails = $this->shopModel->getShop($userData->user_id);
            //  print_r(json_encode($shopDetails));
            // exit;
            $loginData = $this->getData();
            $str = date('ymdhms');
            $productID = "product".md5($str);
            $data = [
                'product_name' => ($loginData['product_name']),
                'product_desc' => ($loginData['product_desc']),
                
                'product_cat' => ($loginData['product_cat']),
                'uni' => $shopDetails->uni_name,
                'amount' => ($loginData['amount']),
                'seller_id' => $userData->user_id,
                'shop_id' => $userData->shop_id,
                'product_id' => $productID,
                'product_name_err' => '',
                'product_desc_err' => '',
                
                'amount_err' => '',
                'product_cat_err' => '',
                'product_img1_err' => '',
                'product_img2_err' => '',
                'product_img3_err' => '',
                'product_img4_err' => '',
                'product_img5_err' => '',
                'seller_id_err' => '',
                'shop_id_err' => '',
                'product_id_err' => ''
            ];
            

            // Validate product_name
            if (empty($data['product_name'])) {
                $data['product_name_err'] = 'Please enter the product name';
            }

            // Validate product_desc
            if (empty($data['product_desc'])) {
                $data['product_desc_err'] = 'Please enter the product description';
            }

        
            if (empty($data['amount'])) {
                $data['amount_err'] = 'Please enter the shop name';
            }

            // Validate product_cat
            if (empty($data['product_cat'])) {
                $data['product_cat_err'] = 'Please enter the product category';
            }

$data['product_img1'] = isset($loginData['product_img1']) && !empty($loginData['product_img1']) ? $loginData['product_img1'] : '0';
$data['product_img2'] = isset($loginData['product_img2']) && !empty($loginData['product_img2']) ? $loginData['product_img2'] : '0';
$data['product_img3'] = isset($loginData['product_img3']) && !empty($loginData['product_img3']) ? $loginData['product_img3'] : '0';
$data['product_img4'] = isset($loginData['product_img4']) && !empty($loginData['product_img4']) ? $loginData['product_img4'] : '0';
$data['product_img5'] = isset($loginData['product_img5']) && !empty($loginData['product_img5']) ? $loginData['product_img5'] : '0';

// Check if all product images are empty after the assignment
if ($data['product_img1'] === '0' && $data['product_img2'] === '0' && $data['product_img3'] === '0' && $data['product_img4'] === '0' && $data['product_img5'] === '0') {
    $data['product_img1_err'] = 'Please upload at least one product image';
}
   
            // Validate seller_id
            if (empty($data['seller_id'])) {
                $data['seller_id_err'] = 'Seller ID is required';
            }

            // Validate shop_id
            if (empty($data['shop_id'])) {
                $data['shop_id_err'] = 'Shop ID is required';
            }

            // Validate product_id
            if (empty($data['product_id'])) {
                $data['product_id_err'] = 'Product ID is required';
            }

            if (empty($data['product_name_err']) &&
            empty($data['product_desc_err']) &&
            empty($data['product_cat_err']) &&
            empty($data['product_img1_err']) &&
   
            empty($data['seller_id_err']) &&
            empty($data['shop_id_err']) &&
            empty($data['amount_err']) &&
            empty($data['product_id_err'])
     ) {
      if ($this->productModel->createProduct($data)) {
                    $response = [
                        'status' => true,
                        'message' => 'product created'
                  ];
                  
                } else {
                  $response = [
                        'status' => false,
                        'message' => 'failed to create product'
                  ];
                 
                }
        
        print_r(json_encode($response));
        exit;

            } else {
                $response = [
                    'status' => false,
                    'message' => 'missing param',
                    'data' => $data
               ];
               print_r(json_encode($response));
               exit;
            }
        } else {
            $response = [
                'status' => false,
                'message' => 'wrong method'
           ];
           print_r(json_encode($response));
           exit;
        }
    }
    
    
    public function addToWishList()
    {
        try {
            $userData = $this->RouteProtection();
        } catch (UnexpectedValueException $e) {
            $res = [
                'status' => 401,
                'message' =>  $e->getMessage(),
            ];
            print_r(json_encode($res));
            exit;
        }
        if ($_SERVER['REQUEST_METHOD'] == 'POST') {
             $loginDatax = $this->getData();
             $productID = $loginDatax['product_id'];
        $shopDetails = $this->productModel->getProductById2($productID);
            $loginData = $shopDetails;
            $data = [
                'product_name' => $loginData->product_name,
                'product_desc' => $loginData->product_desc,
               
                'product_cat' => $loginData->product_cat,
               
                'user_id' => $userData->user_id,
                'seller_id' => $loginData->seller_id,
                'shop_id' => $userData->shop_id,
                'product_id' => $loginData->product_id,
                'amount' => $loginData->amount,
                'product_name_err' => '',
                'product_desc_err' => '',
                
                'product_cat_err' => '',
                'product_img1_err' => '',
                'product_img2_err' => '',
                'product_img3_err' => '',
                'product_img4_err' => '',
                'product_img5_err' => '',
                'seller_id_err' => '',
                'shop_id_err' => '',
                'product_id_err' => ''
            ];
            // print_r(json_encode($data));exit;

            // Validate product_name
            if (empty($data['product_name'])) {
                $data['product_name_err'] = 'Please enter the product name';
            }

            // Validate product_desc
            if (empty($data['product_desc'])) {
                $data['product_desc_err'] = 'Please enter the product description';
            }

            // Validate shop_name
         
            // Validate product_cat
            if (empty($data['product_cat'])) {
                $data['product_cat_err'] = 'Please enter the product category';
            }

          $data['product_img1'] = isset($loginData['product_img1']) && !empty($loginData['product_img1']) ? $loginData['product_img1'] : '0';
$data['product_img2'] = isset($loginData['product_img2']) && !empty($loginData['product_img2']) ? $loginData['product_img2'] : '0';
$data['product_img3'] = isset($loginData['product_img3']) && !empty($loginData['product_img3']) ? $loginData['product_img3'] : '0';
$data['product_img4'] = isset($loginData['product_img4']) && !empty($loginData['product_img4']) ? $loginData['product_img4'] : '0';
$data['product_img5'] = isset($loginData['product_img5']) && !empty($loginData['product_img5']) ? $loginData['product_img5'] : '0';

// Check if all product images are empty after the assignment
if ($data['product_img1'] === '0' && $data['product_img2'] === '0' && $data['product_img3'] === '0' && $data['product_img4'] === '0' && $data['product_img5'] === '0') {
    $data['product_img1_err'] = 'Please upload at least one product image';
}
   

            // Validate seller_id
            if (empty($data['seller_id'])) {
                $data['seller_id_err'] = 'Seller ID is required';
            }

            // Validate shop_id
            if (empty($data['shop_id'])) {
                $data['shop_id_err'] = 'Shop ID is required';
            }

            // Validate product_id
            if (empty($data['product_id'])) {
                $data['product_id_err'] = 'Product ID is required';
            }

            if (empty($data['product_name_err']) &&
            empty($data['product_desc_err']) &&
            empty($data['product_cat_err']) &&
            empty($data['product_img1_err']) &&
            empty($data['product_img2_err']) &&
            empty($data['product_img3_err']) &&
            empty($data['product_img4_err']) &&
            empty($data['product_img5_err']) &&
            empty($data['seller_id_err']) &&
            empty($data['shop_id_err']) &&
            empty($data['product_id_err'])
     ) {
      if ($this->productModel->addToWishList($data)) {
                    $response = [
                        'status' => true,
                        'message' => 'product added to wishlist'
                  ];
                  
                } else {
                  $response = [
                        'status' => false,
                        'message' => 'failed to create product'
                  ];
                 
                }
        
        print_r(json_encode($response));
        exit;
        
                        
                
                
                
                
            } else {
                $response = [
                    'status' => false,
                    'message' => 'missing param',
                    'data' => $data
               ];
               print_r(json_encode($response));
               exit;
            }
        } else {
            $response = [
                'status' => false,
                'message' => 'wrong method'
           ];
           print_r(json_encode($response));
           exit;
        }
    }
    public function createProduct2()
    {
        try {
            $userData = $this->RouteProtection();
        } catch (UnexpectedValueException $e) {
            $res = [
                'status' => 401,
                'message' =>  $e->getMessage(),
            ];
            print_r(json_encode($res));
            exit;
        }
        if ($_SERVER['REQUEST_METHOD'] == 'POST') {
             $shopDetails = $this->shopModel->getShop($userData->user_id);
 
            $loginData = $this->getData();
            $str = date('ymdhms');
            $productID = "product".md5($str);
            $data = [
                'product_name' => ($loginData['product_name']),
                'product_desc' => ($loginData['product_desc']),
                'amount' => ($loginData['amount']),
                'product_cat' => ($loginData['product_cat']),
              'uni' => $shopDetails->shop_location,
                'seller_id' => $userData->user_id,
                'shop_id' => $userData->shop_id,
                'product_id' => $productID,
                'product_name_err' => '',
                'product_desc_err' => '',
                'product_cat_err' => '',
                'product_img1_err' => '',
                'product_img2_err' => '',
                'product_img3_err' => '',
                'product_img4_err' => '',
                'product_img5_err' => '',
                'seller_id_err' => '',
                'shop_id_err' => '',
                'product_id_err' => ''
            ];
            

            // Validate product_name
            if (empty($data['product_name'])) {
                $data['product_name_err'] = 'Please enter the product name';
            }

            // Validate product_desc
            if (empty($data['product_desc'])) {
                $data['product_desc_err'] = 'Please enter the product description';
            }
            // Validate product_cat
            if (empty($data['product_cat'])) {
                $data['product_cat_err'] = 'Please enter the product category';
            }

            // Validate product images (at least one required)
      
          $data['product_img1'] = isset($loginData['product_img1']) && !empty($loginData['product_img1']) ? $loginData['product_img1'] : '0';
$data['product_img2'] = isset($loginData['product_img2']) && !empty($loginData['product_img2']) ? $loginData['product_img2'] : '0';
$data['product_img3'] = isset($loginData['product_img3']) && !empty($loginData['product_img3']) ? $loginData['product_img3'] : '0';
$data['product_img4'] = isset($loginData['product_img4']) && !empty($loginData['product_img4']) ? $loginData['product_img4'] : '0';
$data['product_img5'] = isset($loginData['product_img5']) && !empty($loginData['product_img5']) ? $loginData['product_img5'] : '0';

// Check if all product images are empty after the assignment
if ($data['product_img1'] === '0' && $data['product_img2'] === '0' && $data['product_img3'] === '0' && $data['product_img4'] === '0' && $data['product_img5'] === '0') {
    $data['product_img1_err'] = 'Please upload at least one product image';
}
            // Validate seller_id
            if (empty($data['seller_id'])) {
                $data['seller_id_err'] = 'Seller ID is required';
            }

            // Validate shop_id
            if (empty($data['shop_id'])) {
                $data['shop_id_err'] = 'Shop ID is required';
            }

            // Validate product_id
            if (empty($data['product_id'])) {
                $data['product_id_err'] = 'Product ID is required';
            }

            if (empty($data['product_name_err']) &&
            empty($data['product_desc_err']) &&
            empty($data['shop_name_err']) &&
            empty($data['product_cat_err']) &&
            empty($data['product_img1_err']) ||
            empty($data['product_img2_err']) ||
            empty($data['product_img3_err']) ||
            empty($data['product_img4_err']) ||
            empty($data['product_img5_err']) ||
            empty($data['seller_id_err']) &&
            empty($data['shop_id_err']) &&
            empty($data['product_id_err'])
     ) {
         
         
         
                
                
                
                // Initialize an array to store the new image names after upload
$new_image_names = [];

$extensions = ["jpeg", "png", "jpg"];
$types = ["image/jpeg", "image/jpg", "image/png"];

// List of image fields to process
$image_fields = ['product_img1', 'product_img2', 'product_img3', 'product_img4', 'product_img5'];

foreach ($image_fields as $image_field) {
    if (isset($data[$image_field])) {
        $img_name = $data[$image_field]['name'];
        $img_type = $data[$image_field]['type'];
        $tmp_name = $data[$image_field]['tmp_name'];

        // Validate file extension
        $img_explode = explode('.', $img_name);
        $img_ext = end($img_explode);
        
        if (in_array($img_ext, $extensions) === true) {
            // Validate file type
            if (in_array($img_type, $types) === true) {
                $time = time();
                $new_img_name = $time . "_" . $img_name; // Unique name for the image

                // Attempt to move the uploaded file
                if (move_uploaded_file($tmp_name, ASSETS . "/img/products/" . $new_img_name)) {
                    $new_image_names[$image_field] = $new_img_name;
                } else {
                    $response = array(
                        'status' => 'false',
                        'message' => "Upload failed for $image_field",
                    );
                    print_r(json_encode($response));
                    exit;
                }
            } else {
                $response = array(
                    'status' => 'false',
                    'message' => "Invalid file type for $image_field. Allowed types are: " . implode(', ', $types),
                );
                print_r(json_encode($response));
                exit;
            }
        } else {
            $response = array(
                'status' => 'false',
                'message' => "Invalid file extension for $image_field. Allowed extensions are: " . implode(', ', $extensions),
            );
            print_r(json_encode($response));
            exit;
        }
    } else {
        $response = array(
            'status' => 'false',
            'message' => "$image_field not set",
        );
        print_r(json_encode($response));
        exit;
    }
    
}

foreach ($new_image_names as $key => $value) {
    $data[$key] = $value;
}

        // After validating and uploading all images, process the rest of the form data
      if ($this->productModel->createProduct($data)) {
                    $response = [
                        'status' => true,
                        'message' => 'product created'
                  ];
                  
                } else {
                  $response = [
                        'status' => false,
                        'message' => 'failed to create product'
                  ];
                 
                }
        
        print_r(json_encode($response));
        exit;
        
                        
                
                
                
                
            } else {
                $response = [
                    'status' => false,
                    'message' => 'missing param',
                    'data' => $data
               ];
               print_r(json_encode($response));
               exit;
            }
        } else {
            $response = [
                'status' => false,
                'message' => 'wrong method'
           ];
           print_r(json_encode($response));
           exit;
        }
    }
    public function createFoodProduct()
    {
        try {
            $userData = $this->RouteProtection();
        } catch (UnexpectedValueException $e) {
            $res = [
                'status' => 401,
                'message' =>  $e->getMessage(),
            ];
            print_r(json_encode($res));
            exit;
        }
        if ($_SERVER['REQUEST_METHOD'] == 'POST') {
            $loginData = $this->getData();
            $str = date('ymdhms');
            $productID = "product".md5($str);
            $data = [
                'product_name' => ($loginData['product_name']),
                'product_desc' => ($loginData['product_desc']),
          
                'amount' => ($loginData['amount']),
                'product_cat' => ($loginData['product_cat']),
                'product_img1' => $_FILES['product_img1'],
                'product_img2' => $_FILES['product_img2'],
                'product_img3' => $_FILES['product_img3'],
                'product_img4' => $_FILES['product_img4'],
                'product_img5' => $_FILES['product_img5'],
                'seller_id' => $userData->user_id,
                'shop_id' => $userData->shop_id,
                'product_id' => $productID,
                'product_name_err' => '',
                'product_desc_err' => '',
              
                'product_cat_err' => '',
                'product_img1_err' => '',
                'product_img2_err' => '',
                'product_img3_err' => '',
                'product_img4_err' => '',
                'product_img5_err' => '',
                'seller_id_err' => '',
                'shop_id_err' => '',
                'product_id_err' => ''
            ];
            

            // Validate product_name
            if (empty($data['product_name'])) {
                $data['product_name_err'] = 'Please enter the product name';
            }

            // Validate product_desc
            if (empty($data['product_desc'])) {
                $data['product_desc_err'] = 'Please enter the product description';
            }

 

            // Validate product_cat
            if (empty($data['product_cat'])) {
                $data['product_cat_err'] = 'Please enter the product category';
            }

            // Validate product images (at least one required)
            if (empty($data['product_img1']) || empty($data['product_img2']) || empty($data['product_img3']) || empty($data['product_img4']) || empty($data['product_img5'])) {
                $data['product_img1_err'] = 'Please upload at least one product image';
            }

            // Validate seller_id
            if (empty($data['seller_id'])) {
                $data['seller_id_err'] = 'Seller ID is required';
            }

            // Validate shop_id
            if (empty($data['shop_id'])) {
                $data['shop_id_err'] = 'Shop ID is required';
            }

            // Validate product_id
            if (empty($data['product_id'])) {
                $data['product_id_err'] = 'Product ID is required';
            }

            if (empty($data['product_name_err']) &&
            empty($data['product_desc_err']) &&
            empty($data['product_cat_err']) &&
            empty($data['product_img1_err']) &&
            empty($data['product_img2_err']) &&
            empty($data['product_img3_err']) &&
            empty($data['product_img4_err']) &&
            empty($data['product_img5_err']) &&
            empty($data['seller_id_err']) &&
            empty($data['shop_id_err']) &&
            empty($data['product_id_err'])
     ) {
         
         
         
                
                
                
                // Initialize an array to store the new image names after upload
$new_image_names = [];

$extensions = ["jpeg", "png", "jpg"];
$types = ["image/jpeg", "image/jpg", "image/png"];

// List of image fields to process
$image_fields = ['product_img1', 'product_img2', 'product_img3', 'product_img4', 'product_img5'];

foreach ($image_fields as $image_field) {
    if (isset($data[$image_field])) {
        $img_name = $data[$image_field]['name'];
        $img_type = $data[$image_field]['type'];
        $tmp_name = $data[$image_field]['tmp_name'];

        // Validate file extension
        $img_explode = explode('.', $img_name);
        $img_ext = end($img_explode);
        
        if (in_array($img_ext, $extensions) === true) {
            // Validate file type
            if (in_array($img_type, $types) === true) {
                $time = time();
                $new_img_name = $time . "_" . $img_name; // Unique name for the image

                // Attempt to move the uploaded file
                if (move_uploaded_file($tmp_name, ASSETS . "/img/products/" . $new_img_name)) {
                    $new_image_names[$image_field] = $new_img_name;
                } else {
                    $response = array(
                        'status' => 'false',
                        'message' => "Upload failed for $image_field",
                    );
                    print_r(json_encode($response));
                    exit;
                }
            } else {
                $response = array(
                    'status' => 'false',
                    'message' => "Invalid file type for $image_field. Allowed types are: " . implode(', ', $types),
                );
                print_r(json_encode($response));
                exit;
            }
        } else {
            $response = array(
                'status' => 'false',
                'message' => "Invalid file extension for $image_field. Allowed extensions are: " . implode(', ', $extensions),
            );
            print_r(json_encode($response));
            exit;
        }
    } else {
        $response = array(
            'status' => 'false',
            'message' => "$image_field not set",
        );
        print_r(json_encode($response));
        exit;
    }
    
}

foreach ($new_image_names as $key => $value) {
    $data[$key] = $value;
}

      if ($this->productModel->createFoodProduct($data)) {
                    $response = [
                        'status' => true,
                        'message' => 'food product created'
                  ];
                  
                } else {
                  $response = [
                        'status' => false,
                        'message' => 'failed to create product'
                  ];
                 
                }
        
        print_r(json_encode($response));
        exit;
        
                        
                
                
                
                
            } else {
                $response = [
                    'status' => false,
                    'message' => 'missing param',
                    'data' => $data
               ];
               print_r(json_encode($response));
               exit;
            }
        } else {
            $response = [
                'status' => false,
                'message' => 'wrong method'
           ];
           print_r(json_encode($response));
           exit;
        }
    }
    public function editProduct()
    {
        try {
            $userData = $this->RouteProtection();
        } catch (UnexpectedValueException $e) {
            $res = [
                'status' => 401,
                'message' =>  $e->getMessage(),
            ];
            print_r(json_encode($res));
            exit;
        }
        if ($_SERVER['REQUEST_METHOD'] == 'POST') {
            $loginData = $this->getData();
            $str = date('ymdhms');
            $productID = "product".md5($str);
            $data = [
                'product_name' => ($loginData['product_name']),
                'product_desc' => ($loginData['product_desc']),
                'product_cat' => ($loginData['product_cat']),
              'product_img1' =>($loginData['product_img1']),
                'product_img2' => ($loginData['product_img2']),
                'product_img3' => ($loginData['product_img3']),
                'product_img4' => ($loginData['product_img4']),
                'product_img5' => ($loginData['product_img5']),
                'seller_id' => $userData->user_id,
                'shop_id' => $userData->shop_id,
                'product_id' => ($loginData['product_id']),
                'amount' => ($loginData['amount']),
                'product_name_err' => '',
                'product_desc_err' => '',
                'product_cat_err' => '',
                'product_img1_err' => '',
                'product_img2_err' => '',
                'product_img3_err' => '',
                'product_img4_err' => '',
                'product_img5_err' => '',
                'seller_id_err' => '',
                'shop_id_err' => '',
                'product_id_err' => ''
            ];
            

            // Validate product_name
            if (empty($data['product_name'])) {
                $data['product_name_err'] = 'Please enter the product name';
            }

            // Validate product_desc
            if (empty($data['product_desc'])) {
                $data['product_desc_err'] = 'Please enter the product description';
            }
            // Validate product_cat
            if (empty($data['product_cat'])) {
                $data['product_cat_err'] = 'Please enter the product category';
            }

            // Validate product images (at least one required)
            if (empty($data['product_img1']) && empty($data['product_img2']) && empty($data['product_img3']) && empty($data['product_img4']) && empty($data['product_img5'])) {
                $data['product_img1_err'] = 'Please upload at least one product image';
            }

            // Validate seller_id
            if (empty($data['seller_id'])) {
                $data['seller_id_err'] = 'Seller ID is required';
            }

            // Validate shop_id
            if (empty($data['shop_id'])) {
                $data['shop_id_err'] = 'Shop ID is required';
            }

            // Validate product_id
            if (empty($data['product_id'])) {
                $data['product_id_err'] = 'Product ID is required';
            }

            if (empty($data['product_name_err']) &&
            empty($data['product_desc_err']) &&
            empty($data['product_cat_err']) &&
            empty($data['product_img1_err']) &&
            empty($data['product_img2_err']) &&
            empty($data['product_img3_err']) &&
            empty($data['product_img4_err']) &&
            empty($data['product_img5_err']) &&
            empty($data['seller_id_err']) &&
            empty($data['shop_id_err']) &&
            empty($data['product_id_err'])
     ) {
                     
            
                if ($this->productModel->editProduct($data)) {
                    $response = [
                        'status' => true,
                        'message' => 'product created'
                   ];
                   print_r(json_encode($response));
                   exit;
                } else {
                   $response = [
                        'status' => false,
                        'message' => 'failed to create product'
                   ];
                   print_r(json_encode($response));
                   exit;
                }
            } else {
                $response = [
                    'status' => false,
                    'message' => 'missing param',
                    'data' => $data
               ];
               print_r(json_encode($response));
               exit;
            }
        } else {
            $response = [
                'status' => false,
                'message' => 'wrong method'
           ];
           print_r(json_encode($response));
           exit;
        }
    }
    public function editProduct2()
    {
        try {
            $userData = $this->RouteProtection();
        } catch (UnexpectedValueException $e) {
            $res = [
                'status' => 401,
                'message' =>  $e->getMessage(),
            ];
            print_r(json_encode($res));
            exit;
        }
        if ($_SERVER['REQUEST_METHOD'] == 'POST') {
            $loginData = $this->getData();
            $str = date('ymdhms');
            $productID = "product".md5($str);
            $data = [
                'product_name' => ($loginData['product_name']),
                'product_desc' => ($loginData['product_desc']),
         
                'product_cat' => ($loginData['product_cat']),
                'amount' => ($loginData['amount']),
                'product_img1' => $_FILES['product_img1'],
                'product_img2' => $_FILES['product_img2'],
                'product_img3' => $_FILES['product_img3'],
                'product_img4' => $_FILES['product_img4'],
                'product_img5' => $_FILES['product_img5'],
                'seller_id' => $userData->user_id,
                'shop_id' => $userData->shop_id,
                'product_id' => ($loginData['product_id']),
                'product_name_err' => '',
                'product_desc_err' => '',
                'product_cat_err' => '',
                'product_img1_err' => '',
                'product_img2_err' => '',
                'product_img3_err' => '',
                'product_img4_err' => '',
                'product_img5_err' => '',
                'seller_id_err' => '',
                'shop_id_err' => '',
                'product_id_err' => ''
            ];
            

            // Validate product_name
            if (empty($data['product_name'])) {
                $data['product_name_err'] = 'Please enter the product name';
            }

            // Validate product_desc
            if (empty($data['product_desc'])) {
                $data['product_desc_err'] = 'Please enter the product description';
            }
            // Validate product_cat
            if (empty($data['product_cat'])) {
                $data['product_cat_err'] = 'Please enter the product category';
            }

            // Validate product images (at least one required)
            if (empty($data['product_img1']) && empty($data['product_img2']) && empty($data['product_img3']) && empty($data['product_img4']) && empty($data['product_img5'])) {
                $data['product_img1_err'] = 'Please upload at least one product image';
            }

            // Validate seller_id
            if (empty($data['seller_id'])) {
                $data['seller_id_err'] = 'Seller ID is required';
            }

            // Validate shop_id
            if (empty($data['shop_id'])) {
                $data['shop_id_err'] = 'Shop ID is required';
            }

            // Validate product_id
            if (empty($data['product_id'])) {
                $data['product_id_err'] = 'Product ID is required';
            }

            if (empty($data['product_name_err']) &&
            empty($data['product_desc_err']) &&
            empty($data['product_cat_err']) &&
            empty($data['product_img1_err']) &&
            empty($data['product_img2_err']) &&
            empty($data['product_img3_err']) &&
            empty($data['product_img4_err']) &&
            empty($data['product_img5_err']) &&
            empty($data['seller_id_err']) &&
            empty($data['shop_id_err']) &&
            empty($data['product_id_err'])
     ) {
                     
                
                // Initialize an array to store the new image names after upload
$new_image_names = [];

$extensions = ["jpeg", "png", "jpg"];
$types = ["image/jpeg", "image/jpg", "image/png"];

// List of image fields to process
$image_fields = ['product_img1', 'product_img2', 'product_img3', 'product_img4', 'product_img5'];

foreach ($image_fields as $image_field) {
    if (isset($data[$image_field])) {
        $img_name = $data[$image_field]['name'];
        $img_type = $data[$image_field]['type'];
        $tmp_name = $data[$image_field]['tmp_name'];

        // Validate file extension
        $img_explode = explode('.', $img_name);
        $img_ext = end($img_explode);
        
        if (in_array($img_ext, $extensions) === true) {
            // Validate file type
            if (in_array($img_type, $types) === true) {
                $time = time();
                $new_img_name = $time . "_" . $img_name; // Unique name for the image

                // Attempt to move the uploaded file
                if (move_uploaded_file($tmp_name, ASSETS . "/img/products/" . $new_img_name)) {
                    $new_image_names[$image_field] = URLROOT . $new_img_name;
                } else {
                    $response = array(
                        'status' => 'false',
                        'message' => "Upload failed for $image_field",
                    );
                    print_r(json_encode($response));
                    exit;
                }
            } else {
                $response = array(
                    'status' => 'false',
                    'message' => "Invalid file type for $image_field. Allowed types are: " . implode(', ', $types),
                );
                print_r(json_encode($response));
                exit;
            }
        } else {
            $response = array(
                'status' => 'false',
                'message' => "Invalid file extension for $image_field. Allowed extensions are: " . implode(', ', $extensions),
            );
            print_r(json_encode($response));
            exit;
        }
    } else {
        $response = array(
            'status' => 'false',
            'message' => "$image_field not set",
        );
        print_r(json_encode($response));
        exit;
    }
    
}


foreach ($new_image_names as $key => $value) {
    $data[$key] = $value;
}

        // After validating and uploading all images, process the rest of the 
                if ($this->productModel->editProduct($data)) {
                    $response = [
                        'status' => true,
                        'message' => 'product created'
                   ];
                   print_r(json_encode($response));
                   exit;
                } else {
                   $response = [
                        'status' => false,
                        'message' => 'failed to create product'
                   ];
                   print_r(json_encode($response));
                   exit;
                }
            } else {
                $response = [
                    'status' => false,
                    'message' => 'missing param',
                    'data' => $data
               ];
               print_r(json_encode($response));
               exit;
            }
        } else {
            $response = [
                'status' => false,
                'message' => 'wrong method'
           ];
           print_r(json_encode($response));
           exit;
        }
    }
    public function ProductById()
    {
        try {
            $userData = $this->RouteProtection();
        } catch (UnexpectedValueException $e) {
            $res = [
                'status' => 401,
                'message' =>  $e->getMessage(),
            ];
            print_r(json_encode($res));
            exit;
        }
        
        $loginData = $this->getData();
        $productID = $loginData['product_id'];
        $Details = $this->productModel->getProductReviews($productID);
         
        $shopDetails = $this->productModel->getProductById( $productID);
         $shopDetailsxx = $this->shopModel->getShop($shopDetails->seller_id);

          if ($shopDetails != null) {
              shuffle($Details);
                    $response = [
                        'status' => true,
                        'data' => $shopDetails,
                        'shop' => $shopDetailsxx,
                        'reviews' => $Details,
                   ];
                   print_r(json_encode($response));
                   exit;
                } else {
                   $response = [
                        'status' => false,
                        'data' => []                   ];
                   print_r(json_encode($response));
                   exit;
                }
      
    }
    public function getProductByShop()
    {
        try {
            $userData = $this->RouteProtection();
        } catch (UnexpectedValueException $e) {
            $res = [
                'status' => 401,
                'message' =>  $e->getMessage(),
            ];
            print_r(json_encode($res));
            exit;
        }
        $products = $this->productModel->getProducts($userData->user_id, $userData->shop_id);
          if ($products) {
                    $response = [
                        'status' => true,
                        'data' => $products
                   ];
                   print_r(json_encode($response));
                   exit;
                } else {
                   $response = [
                        'status' => false,
                        'message' => 'failed to get product by shop'
                   ];
                   print_r(json_encode($response));
                   exit;
                }
      
    }
    
   public function getProductByUni()
{
    $data = $this->getData();
    $uni = $data['university'];
    $Details = $this->productModel->getProductByUni($uni);

    $products = []; // Initialize an empty array for products

    if (!empty($Details)) {
        foreach ($Details as $Details_item) {
          
            
            $reviews = $this->productModel->getProductReviews($Details_item->product_id);
            
            $products[] = [
                'details' => $Details_item,
                'reviews' => $reviews
            ];
        }

        // Shuffle products if there are any and prepare success response
        if (!empty($products)) {
            shuffle($products);
            $response = [
                'status' => true,
                'data' => $products
            ];
        } else {
            // Case when products array is empty even if $Details is not empty
            $response = [
                'status' => false,
                'message' => 'No products found for this university'
            ];
        }
    } else {
        // Case when $Details is empty
        $response = [
            'status' => false,
            'message' => 'No products found for this university'
        ];
    }

    print_r(json_encode($response));
    exit;
}

      public function getProductByUniCat()
    {
        
      $data = $this->getData();
      $uni = $data['university'];
      $cat = $data['category'];
        $Details = $this->productModel->getProductByUniCat($uni, $cat);
        $products = [];
        foreach($Details as $Details_item){
            
             
            $reviews = $this->productModel->getProductReviews($Details_item->product_id);
            
             $products[] = [
                    'details' => $Details_item,
                    'reviews' => $reviews
                 ];
        }
        
         if ($Details_item) {
             shuffle($products);
                    $response = [
                        'status' => true,
                        'data' => $products
                   ];
                   print_r(json_encode($response));
                   exit;
                } else {
                   $response = [
                        'status' => false,
                        'message' => 'failed to get all product'
                   ];
                   print_r(json_encode($response));
                   exit;
                }
      
      
    }
    public function getAllProducts()
    {
  
        $Details = $this->productModel->getAllProducts();
        
       
        $products = [];
        foreach($Details as $Details_item){
            
             
            $reviews = $this->productModel->getProductReviews($Details_item->product_id);
            
             $products[] = [
                    'details' => $Details_item,
                    'reviews' => $reviews
                 ];
        }
        
         if ($Details_item) {
             shuffle($products);
                    $response = [
                        'status' => true,
                        'data' => $products
                   ];
                   print_r(json_encode($response));
                   exit;
                } else {
                   $response = [
                        'status' => false,
                        'message' => 'failed to get all product'
                   ];
                   print_r(json_encode($response));
                   exit;
                }
      
    }
    
    public function getProductReviews()
    {
        
  
        $loginData = $this->getData();
        $productID = $loginData['product_id'];
          if(empty($productID)){
                $res = json_encode([
                        'status' => false,
                        'message' => 'baba rest'
                    ]);
                    print_r($res);
                    exit;
           }
        $Details = $this->productModel->getProductReviews($productID);
        
          if ($Details) {
              shuffle($Details);
                    $response = [
                        'status' => true,
                        'data' => $Details
                   ];
                   print_r(json_encode($response));
                   exit;
                } else {
                   $response = [
                        'status' => false,
                        'message' => 'failed to get product review'
                   ];
                   print_r(json_encode($response));
                   exit;
                }
      
    }
    public function getAverageReview()
    {
        
  
        $loginData = $this->getData();
        $productID = $loginData['product_id'];
        $Details = $this->productModel->getAverageReview($productID);
        // shuffle($Details);
          if ($Details) {
                    $response = [
                        'status' => true,
                        'data' => $Details
                   ];
                   print_r(json_encode($response));
                   exit;
                } else {
                   $response = [
                        'status' => false,
                        'message' => 'failed to get average review'
                   ];
                   print_r(json_encode($response));
                   exit;
                }
        // print_r(json_encode($Details));
      
    }
    public function activateShop()
    {
        try {
            $userData = $this->RouteProtection();
        } catch (UnexpectedValueException $e) {
            $res = [
                'status' => 401,
                'message' =>  $e->getMessage(),
            ];
            print_r(json_encode($res));
            exit;
        }
        if ($_SERVER['REQUEST_METHOD'] == 'POST') {
            $loginData = $this->getData();
            $str = date('ymdhms');
            $shopID = "user".md5($str);
            $data = [
                'shop_plan' => ($loginData['shop_plan']),
                'plan_price' => ($loginData['plan_price']),
                'expiry_date' => ($loginData['expiry_date']),
                'seller_id' => $userData->user_id,
                'shop_id' => $shopID,
                'shop_plan_err' => '',
                'plan_price_err' => '',
                'expiry_date_err' => ''
            ];


            // Validate shop name
            if (empty($data['shop_plan'])) {
                $data['shop_plan_err'] = 'Please enter the shop name';
            }

            // Validate shop description
            if (empty($data['plan_price'])) {
                $data['plan_price_err'] = 'Please enter the shop description';
            }

            // Validate shop WhatsApp link
            if (empty($data['expiry_date'])) {
                $data['expiry_date_err'] = 'Please enter the shop WhatsApp link';
            }


            // Make sure there are no errors
            if (
                empty($data['shop_plan_err']) &&
                empty($data['plan_price_err']) &&
                empty($data['expiry_date_err'])
            ) {
                // Validation passed
                //Execute
                if ($this->shopModel->activateShop($data)) {
                    $response = [
                        'status' => true,
                        'message' => 'Shop activated'
                   ];
                   print_r(json_encode($response));
                   exit;
                } else {
                   $response = [
                        'status' => false,
                        'message' => 'failed to activate shop'
                   ];
                   print_r(json_encode($response));
                   exit;
                }
            } else {
                $response = [
                    'status' => false,
                    'message' => 'missing param',
                    'data' => $data
               ];
               print_r(json_encode($response));
               exit;
            }
        } else {
            $response = [
                'status' => false,
                'message' => 'wrong method'
           ];
           print_r(json_encode($response));
           exit;
        }
    }

  

    // Delete Post
    public function postReview()
    {
        if ($_SERVER['REQUEST_METHOD'] == 'POST') {
             try {
            $userData = $this->RouteProtection();
        } catch (UnexpectedValueException $e) {
            $res = [
                'status' => 401,
                'message' =>  $e->getMessage(),
            ];
            print_r(json_encode($res));
            exit;
        }
            
           $data = $this->getData();
           $productID = $data['product_id'];
           $rating = $data['rating'];
           $review = $data['review'];
           
           if(empty($productID) || empty($rating) || empty($review)){
                $res = json_encode([
                        'status' => false,
                        'message' => 'baba rest'
                    ]);
                    print_r($res);
                    exit;
           }
           $dat = [
               'product_id' => $productID,
               'rating' => $rating,
               'desc' => $review,
               'username' =>  $userData->username,
               'img' => $userData->imageUrl,
               'user_id' => $userData->user_id
               ];
           if ($this->productModel->postReviewxx($dat))
           {
                 $res = json_encode([
                        'status' => true,
                        'message' => 'review posted'
                    ]);
                    print_r($res);
           }else{
                 $res = json_encode([
                        'status' => false,
                        'message' => 'failed'
                    ]);
                    print_r($res);
           }
           
          
        } else {
               $res = json_encode([
                        'status' => false,
                        'message' => 'wrong method'
                    ]);
                    print_r($res);
        }
    }
    
    

    // Delete Post
    public function deleteProduct()
    {
        if ($_SERVER['REQUEST_METHOD'] == 'POST') {
             try {
            $userData = $this->RouteProtection();
        } catch (UnexpectedValueException $e) {
            $res = [
                'status' => 401,
                'message' =>  $e->getMessage(),
            ];
            print_r(json_encode($res));
            exit;
        }
            
           $data = $this->getData();
           $productID = $data['product_id'];
           $product =  $this->productModel->getProductById2($productID);
           
           if($userData->user_id == $product->seller_id ){
               $id = $product->id;
           
            if ($this->productModel->deleteProduct($id)) {
                $res = json_encode([
                        'status' => true,
                        'message' => 'product deleted'
                    ]);
                    print_r($res);
            } else {
                    $res = json_encode([
                        'status' => false,
                        'message' => 'product not deleted'
                    ]);
                    print_r($res); 
            }
           }else{
                 $res = json_encode([
                        'status' => false,
                        'message' => 'this product cannot be deleted by you'
                    ]);
                    print_r($res); 
           }
        } else {
               $res = json_encode([
                        'status' => false,
                        'message' => 'wrong method'
                    ]);
                    print_r($res);
        }
    }
}
